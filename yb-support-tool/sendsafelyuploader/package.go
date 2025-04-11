package sendsafelyuploader

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/docker/go-units"
	"golang.org/x/sync/errgroup"
)

// DefaultChunkSize is 2.5 MB, recommended by sendsafely
var DefaultChunkSize int64 = 2.5 * units.MiB

type Package struct {
	Uploader    *Uploader
	Info        PackageInfo
	URL         string
	Checksum    string
	Files       []*File
	ChunkSize   int64
	Concurrency int
	Retries     uint
	m           sync.Mutex
}

type PackageInfo struct {
	PackageID    string `json:"packageId"`
	PackageCode  string `json:"packageCode"`
	ServerSecret string `json:"serverSecret"`

	SSResponse
}

type packageOption func(*Package)

func WithChunkSize(s int64) packageOption {
	return func(p *Package) {
		p.ChunkSize = s
	}
}

func WithConcurrency(c int) packageOption {
	return func(p *Package) {
		p.Concurrency = c
	}
}

func WithRetries(r uint) packageOption {
	return func(p *Package) {
		p.Retries = r
	}
}

func (p *Package) calculateNumberOfParts(fileSize int64) int64 {
	return int64(math.Ceil(float64(fileSize) / float64(p.ChunkSize)))

}

func (u *Uploader) CreateDropzonePackage(options ...packageOption) (*Package, error) {

	endpoint := "/drop-zone/v2.0/package/"

	p := &Package{
		Uploader:  u,
		ChunkSize: DefaultChunkSize,
	}

	for _, opt := range options {
		opt(p)
	}

	body, err := u.sendRequest(http.MethodPut, endpoint, nil)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(body, &p.Info); err != nil {
		return nil, err
	}

	p.createChecksum()
	return p, nil

}

/*
Given a file object,
1) Get file split parts - # of parts and chunk size
2) get upload URLs for those chunks
3) read the chunks, pipe thru encryption
4) upload the file to the URL
*/
func (p *Package) AddFileToPackage(file *os.File) (*File, error) {
	endpoint := fmt.Sprintf("/drop-zone/v2.0/package/%s/file", p.Info.PackageCode)
	fInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}

	f := &File{
		Info: &FileInfo{
			Name: filepath.Base(file.Name()),
			Size: fInfo.Size(),
		},
		Ptr:      file,
		PartURLs: make([]string, p.calculateNumberOfParts(fInfo.Size())),
	}

	f.UploadInfo = &FileUpload{
		Name:       f.Info.Name,
		UploadType: "DROP_ZONE",
		Part:       1,
		Parts:      len(f.PartURLs),
		Size:       f.Info.Size,
	}
	fileUploadJSON, err := json.Marshal(f.UploadInfo)
	if err != nil {
		return nil, err
	}

	body, err := p.Uploader.sendRequest(http.MethodPut, endpoint, fileUploadJSON, withReqJSONHeader())
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &f.Info)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal server response: '%s' ; got unmarshal error %s", string(body), err)
	}

	if err := p.GetUploadURLs(f); err != nil {
		return nil, err
	}

	f.Status = make(chan int)

	return f, nil

}

// Sets the upload URLs for the file parts in the provided file pointer
// only 25 part urls are returned - so if more than that are required, retry
func (p *Package) GetUploadURLs(file *File) error {
	if p.Info.PackageCode == "" {
		return fmt.Errorf("No package code provided for file: '%s', cannot get list of upload URLs", file.Info.Name)
	}

	endpoint := fmt.Sprintf("/drop-zone/v2.0/package/%s/file/%s/upload-urls", p.Info.PackageCode, file.Info.ID) //u.FileInfo.FileID)

	f := func(part int) error {
		file.UploadInfo.Part = part
		fileUploadJSON, err := json.Marshal(file.UploadInfo)
		if err != nil {
			return err
		}
		body, err := p.Uploader.sendRequest(http.MethodPost, endpoint, fileUploadJSON, withReqJSONHeader())
		if err != nil {
			return err
		}

		uInfo := UploadUrlInfo{}

		err = json.Unmarshal(body, &uInfo)
		if err != nil {
			return err
		}
		if uInfo.Response != "SUCCESS" {
			fmt.Printf("uploadInfoName: %s, Part: %d\n", file.UploadInfo.Name, file.UploadInfo.Part)
			return fmt.Errorf("Failed to get upload URLs: response: '%s'; message: '%s'", uInfo.Response, uInfo.Message)
		}

		for _, u := range uInfo.URLS {
			file.PartURLs[u.Part-1] = u.URL
		}
		return nil
	}

	// calculate number of calls to get
	batchSize := 25
	calls := int(math.Ceil(float64(file.Info.Parts) / float64(batchSize)))
	for i := 0; i < calls; i++ {
		if err := f(1 + i*batchSize); err != nil {
			return err
		}
	}

	return nil

}

func (p *Package) UploadFileParts(file *File) error {
	defer close(file.Status)

	g := new(errgroup.Group)
	g.SetLimit(p.Concurrency)

	r := bufio.NewReader(file.Ptr)
	bytePool := sync.Pool{New: func() interface{} {
		return make([]byte, p.ChunkSize)
	}}
	for _, URL := range file.PartURLs {
		URL := URL
		buff := bytePool.Get().([]byte)
		b := []byte{}
		c, err := r.Read(buff)
		b = append(b, buff[:c]...)

		if err != nil {
			if err == io.EOF {
			} else {
				return err
			}
		}

		g.Go(func() error {
			encryptB, err := p.Encrypt(bytes.NewReader(b))
			if err != nil {
				return err
			}

			err = p.uploadFilePart(encryptB, URL)
			if err != nil {
				return err
			}
			file.Status <- 1
			return nil
		})

	}
	err := g.Wait()
	return err

}

// try up to Package.Retries
func (p *Package) uploadFilePart(part *bytes.Buffer, url string) error {

	if part.Len() == 0 {
		return fmt.Errorf("File part is empty")
	}

	if url == "" {
		return fmt.Errorf("URL is empty")
	}
	f := func() error {
		req, err := http.NewRequest(http.MethodPut, url, part)
		req.Header.Set("ss-request-api", "DROP_ZONE")

		if err != nil {
			return err
		}

		resp, err := p.Uploader.Client.Do(req)

		if err != nil {
			return err
		}

		if resp.StatusCode != http.StatusOK {
			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("Unable to upload file: Bad HTTP Status Code: %d; Unable to read response body", resp.StatusCode)
			}
			return fmt.Errorf("Unable to upload file: Invalid HTTP status response; HTTP Error code %d; request body %s", resp.StatusCode, string(respBody))
		}
		defer resp.Body.Close()
		return nil
	}

	for {
		if err := f(); err != nil {
			p.m.Lock()
			if p.Retries > 0 {
				p.Retries--
				p.m.Unlock()
				continue
			} else {
				p.m.Unlock()
				return err
			}
		}
		break
	}

	return nil
}

func (p *Package) MarkFileComplete(file *File) error {

	endpoint := fmt.Sprintf("/drop-zone/v2.0/package/%s/file/%s/upload-complete", p.Info.PackageCode, file.Info.ID)

	body, err := p.Uploader.sendRequest(http.MethodPost, endpoint, nil, withReqJSONHeader())
	if err != nil {
		return err
	}

	var resp FileInfo
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}
	if resp.Response != "SUCCESS" {
		return fmt.Errorf("Failed to finalize file '%s' in package '%s' with message: %s", file.Info.Name, p.Info.PackageID, resp.Message)
	}

	return nil
}

// FinalizePackage marks the package complete and returns a link to the package,
// or the error if package finalization did not succeed. Sets Package.URL on success
func (p *Package) FinalizePackage() error {

	endpoint := fmt.Sprintf("/drop-zone/v2.0/package/%s/finalize", p.Info.PackageCode)

	type Rb struct {
		Checksum string
	}

	var rb Rb

	rb.Checksum = p.Checksum

	rbJson, err := json.Marshal(rb)
	if err != nil {
		return err
	}

	body, err := p.Uploader.sendRequest(http.MethodPost, endpoint, rbJson, withReqJSONHeader())
	if err != nil {
		return err
	}

	var resp FinalizeInfo
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}

	if resp.Response != "SUCCESS" {
		return fmt.Errorf("Failed to finalize package '%s' with message: %s", p.Info.PackageID, resp.Message)
	}

	p.URL = resp.Message + "#keyCode=" + p.Uploader.ClientSecret
	return nil
}

func (p *Package) SubmitHostedDropzone(name string, email string) error {
	dropzoneData := url.Values{}
	dropzoneData.Set("name", name)
	dropzoneData.Set("email", email)
	dropzoneData.Set("packageCode", p.Info.PackageCode)
	dropzoneData.Set("publicApiKey", p.Uploader.APIKey)

	endpoint := "/auth/json/?action=submitHostedDropzone"

	body, err := p.Uploader.sendRequest(http.MethodPost, endpoint, []byte(dropzoneData.Encode()), withReqFormHeader())
	if err != nil {
		return err
	}

	var dzInfo HostedDropzoneInfo
	err = json.Unmarshal(body, &dzInfo)
	if err != nil {
		return err
	}
	if dzInfo.Success != "true" {
		return fmt.Errorf("Unable to submit Hosted Dropzone with error %s", dzInfo.Message)
	}

	return p.finalizeHostedDropzone(dzInfo)
}

func (p *Package) finalizeHostedDropzone(dzInfo HostedDropzoneInfo) error {
	finalDZData := url.Values{}

	finalDZData.Set("data", dzInfo.Data)
	finalDZData.Set("digest", dzInfo.Digest)
	finalDZData.Set("secureLink", p.URL)

	for _, url := range dzInfo.IntegrationURLs {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer([]byte(finalDZData.Encode())))
		if err != nil {
			return err
		}
		if err := withReqFormHeader()(req); err != nil {
			return err
		}

		resp, err := p.Uploader.Client.Do(req)

		if err != nil {
			return err
		}

		if resp.StatusCode != http.StatusOK {
			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("Unable to upload file: Bad HTTP Status Code: %d; Unable to read response body", resp.StatusCode)
			}
			return fmt.Errorf("Unable to upload file: Invalid HTTP status response; HTTP Error code %d; request body %s", resp.StatusCode, string(respBody))
		}
		defer resp.Body.Close()
	}
	return nil
}
