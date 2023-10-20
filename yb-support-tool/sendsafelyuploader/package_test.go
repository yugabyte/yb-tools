package sendsafelyuploader

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

var (
	samplePackageCode = "sMN0ghoCOkxFiOBHlXhscb1qS3fMmd7sV0012gIRutU"
	samplePackageID   = "527P-FPRP"
)

func getPackage(u *Uploader) *Package {
	return &Package{
		Uploader:  u,
		ChunkSize: DefaultChunkSize,
		Info: PackageInfo{
			PackageID:   samplePackageID,
			PackageCode: samplePackageCode,
		},
	}
}

func TestGetUploadURLs(t *testing.T) {
	validFile := &File{
		Info:       &FileInfo{Name: "VALID", Parts: 2},
		PartURLs:   make([]string, 2),
		UploadInfo: &FileUpload{Name: "VALID", Part: 1},
	}

	validUploadURLObject := UploadUrlInfo{
		URLS: []UploadURL{
			UploadURL{
				Part: 1,
				URL:  `https://sendsafely-us-west-1.s3-accelerate.amazonaws.com/commercial/e93ec274-e586-4f55-8eab-498a8444cf94/f0805497-314d-4eac-ac52-80f4fb40d9eb-1?AWSAccessKeyId=AKIAJNE5FSA2YFQP4BDA&Expires=1680894043&Signature=3R%2FbQJrY1XXObIa5XUvm6ntk3sE%3D`,
			},
			UploadURL{
				Part: 2,
				URL:  `https://sendsafely-us-west-2.s3-accelerate.amazonaws.com/commercial/e93ec274-e586-4f55-8eab-498a8444cf94/f0805497-314d-4eac-ac52-80f4fb40d9eb-1?AWSAccessKeyId=AKIAJNE5FSA2YFQP4BDA&Expires=1680894043&Signature=3R%2FbQJrY1XXObIa5XUvm6ntk3sE%3D`,
			},
		},

		SSResponse: SSResponse{Response: "SUCCESS"}}
	validUploadURLJSON, _ := json.Marshal(validUploadURLObject)

	// check that we hit endpoint "/drop-zone/v2.0/package/%s/file/%s/upload-urls"
	validServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if !strings.Contains(r.URL.Path, "/upload-urls") {
			t.Errorf("Expected to hit path containing /upload-urls, instead got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		// test that r.Body contains upload-urls
		w.Write(validUploadURLJSON)

	}))
	defer validServer.Close()
	u := getUploader(validServer.URL)

	p := Package{Uploader: u, Files: []*File{validFile}}

	// no error if we have valid package code and fileid
	p.Info.PackageCode = samplePackageCode
	validFile.Info.ID = "8c652651-7e0e-4eb8-bb1b-24710bd4ee35"

	err := p.GetUploadURLs(validFile)
	if err != nil {
		t.Errorf("Expected no error with valid package code and fileid, but recieved %s", err)
	}
	// file should have some URLS
	if len(p.Files[0].PartURLs) == 0 {
		t.Errorf("Expected File to have some Part URLS, but it's empty")
	}

	// and we set UploadUrlInfo properly based on response from server for the first URL
	if p.Files[0].PartURLs[0] != validUploadURLObject.URLS[0].URL {
		t.Errorf("Expected to populate UploadUrlInfo with a valid respose, but did not: found %s for first argument but should be %s", p.Files[0].PartURLs[0], validUploadURLObject.URLS[0].URL)
	}

	// and we set UploadUrlInfo properly based on response from server for the second URL
	if p.Files[0].PartURLs[1] != validUploadURLObject.URLS[1].URL {
		t.Errorf("Expected to populate UploadUrlInfo with a valid respose, but did not: found %s for first argument but should be %s", p.Files[0].PartURLs[1], validUploadURLObject.URLS[1].URL)
	}

	// test that if PackageCode is missing in uploader, getUploadURL fails with error
	p.Info.PackageCode = ""
	err = p.GetUploadURLs(validFile)

	if err == nil {
		t.Error("Expected an error from function because PackageCode empty, but instead function has no err")
	}
	// error string should contain filename for reference
	if !strings.Contains(err.Error(), validFile.Info.Name) {
		t.Errorf("Expected failure message to contain filename: '%s', instead got message: '%s'", validFile.Info.Name, err.Error())
	}

	// test that if we get `"RESPONSE": "FAIL"` return, that we get a valid error

	invalidUploadURLInfo, _ := json.Marshal(
		UploadUrlInfo{
			SSResponse: SSResponse{
				Response: "FAIL",
				Message:  "An error occurred.  Error Id {some error code}"},
		})

	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(invalidUploadURLInfo))

	}))
	defer failServer.Close()

	uf := getUploader(failServer.URL)
	pf := Package{Uploader: uf}

	pf.Info.PackageCode = "sMN0ghoCOkxFiOBHlXhscb1qS3fMmd7sV0012gIRutU"
	err = pf.GetUploadURLs(validFile)

	if err == nil {
		t.Error("Expected to get response message from server that we got a FAILURE, instead got not error")
	}
}

func TestUploadFilePart(t *testing.T) {

	var fileChunkSize = DefaultChunkSize

	// generate byte slice to represent the byte slice being uploaded
	filePart := make([]byte, int(fileChunkSize))

	// non-zero file
	rand.Read(filePart)

	// Example:
	/*
		{
		  "uploadUrls": [
		    {
		      "part": 1,
		      "url": "https://sendsafely-us-west-2.s3-accelerate.amazonaws.com/commercial/e93ec274-e586-4f55-8eab-498a8444cf94/8c652651-7e0e-4eb8-bb1b-24710bd4ee35-1?AWSAccessKeyId=AKIAJNE5FSA2YFQP4BDA&Expires=1664069315&Signature=yOAGja0sPBv75MLa2Jf6YfXYom8%3D"
		    },
		    {
		      "part": 2,
		      "url": "https://sendsafely-us-west-2.s3-accelerate.amazonaws.com/commercial/e93ec274-e586-4f55-8eab-498a8444cf94/8c652651-7e0e-4eb8-bb1b-24710bd4ee35-2?AWSAccessKeyId=AKIAJNE5FSA2YFQP4BDA&Expires=1664069315&Signature=xBkpJlOM1OUkoKRwvBf1mPrDlM0%3D"
		    },
		    {
		      "part": 3,
		      "url": "https://sendsafely-us-west-2.s3-accelerate.amazonaws.com/commercial/e93ec274-e586-4f55-8eab-498a8444cf94/8c652651-7e0e-4eb8-bb1b-24710bd4ee35-3?AWSAccessKeyId=AKIAJNE5FSA2YFQP4BDA&Expires=1664069315&Signature=kxfBBSQfE7oLu83%2FwALAmKboeL8%3D"
		    },
		    {
		      "part": 4,
		      "url": "https://sendsafely-us-west-2.s3-accelerate.amazonaws.com/commercial/e93ec274-e586-4f55-8eab-498a8444cf94/8c652651-7e0e-4eb8-bb1b-24710bd4ee35-4?AWSAccessKeyId=AKIAJNE5FSA2YFQP4BDA&Expires=1664069315&Signature=mfnwodVkgMvRSOSVCrRie3yxqcE%3D"
		    }
		  ],
		  "response": "SUCCESS"
		}

	*/

	validPath := "/commercial/e93ec274-e586-4f55-8eab-498a8444cf94/f0805497-314d-4eac-ac52-80f4fb40d9eb-1?AWSAccessKeyId=AKIAJNE5FSA2YFQP4BDA&Expires=1680894043&Signature=3R%2FbQJrY1XXObIa5XUvm6ntk3sE%3D"
	sampleURL := "http://example.com" + validPath
	sURL, _ := url.Parse(sampleURL)

	validServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != sURL.Path {
			t.Errorf("Expected to request '%s', got: %s", sURL.Path, r.URL.Path)
		}
		if r.Header.Get("ss-request-api") != "DROP_ZONE" {
			t.Errorf("Expected ss-request-api header: DROP_ZONE, got: %s", r.Header.Get("ss-api-key"))
		}
		sentBody, _ := io.ReadAll(r.Body)
		if string(sentBody) != string(filePart) {
			t.Errorf("Expected sent and received messages are not the same, but they are")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testOKResponse))

	}))

	validURL := validServer.URL + validPath

	defer validServer.Close()
	u := getUploader(validServer.URL)

	p := &Package{}

	p.Uploader = u

	// valid file part and valid URL returns no error
	err := p.uploadFilePart(bytes.NewBuffer(filePart), validURL)

	if err != nil {
		t.Errorf("Expected no error but instead got %s", err)
	}

	// if I'm given an invalid URL, returns an error
	invalidURL := "1234"
	err = p.uploadFilePart(bytes.NewBuffer(filePart), invalidURL)
	if err == nil {
		t.Errorf("Expected error on invalid URL but did not receive one")
	}

	// if part []byte is size of zero, returns an error
	err = p.uploadFilePart(bytes.NewBuffer([]byte{}), validURL)
	if err == nil {
		t.Errorf("Expected error for empty part but did not receive one")
	}

	// testing non-ok response
	invalidServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(testOKResponse))
	}))
	err = p.uploadFilePart(bytes.NewBuffer(filePart), invalidServer.URL)
	if err == nil {
		t.Errorf("Expected error due to bad http response code, but did not get one")
	}

	// upload file parts retries up to the limit

	// retry counter - the first touch is not a retry
	i := -1
	var maxRetries uint = 7

	retryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i++
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("FAILED"))

	}))
	p.Retries = maxRetries
	p.uploadFilePart(bytes.NewBuffer([]byte("test")), retryServer.URL)

	if i != int(maxRetries) {
		t.Errorf("Expected to retry file upload %d times, but only tried %d", maxRetries, i)
	}

}

func TestSubmitHostedDropzone(t *testing.T) {

	endpointQuery := "action=submitHostedDropzone"
	endpointPath := "/auth/json"

	dropzoneName := "123"
	dropzoneEmail := "test@example.com"

	validFinalizeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":"true"}`))
	}))
	validResponse := fmt.Sprintf(`{"success":"true","data":"0--m2RXSf1KHyqg5i9YnRvKr4CBkJci0WnLuW7ZVx0vpf4QdTsBeuxOa6OAvsuZYEVFgTbDAQWoK7dndVSdByKA_NGKcTogLk326jf1wm88Q0T_c7Nyldd3Ibk2mzENL08wsCUbpTfu0lIxnwqOFb27MVFzI9PhOVUI8cslQbOJ40mL_IGF0Z4m8tsQL2-UeUfycS94IrdMxBqp8CSbhoW6CehGwcm7Co-Ubd6C_k3dsdVcOMk1YmE_typNKfb9ZCP9i0prRADirYxAXu6nsJJ2T2QzTusBMMP2f9ReSORuNxvr1XXIsIG-oAIsa116naVE3LRMLwdJCVyb3HymL6bJ0mRFZUEmKg9On7K3zXJgiwpjFX2V7UwRRbJgJvg6Jp1GTx87DSaAxR3auVcBiEJeID5vggGBgVD3X4FPOVnEUFWpdgu10EH9vifrms-rRBRksqclSFb0guiqRi-zVQWMAo0Vp-RbE_SUy7_k2QUUxnhXXdtHPPdMs5n-RGoGs3EqFnQZMaTMxS7Lj6HjRQaQ5n-HYcIDNH6lWUl35jVhtF_eI6StizIEfURCBYbMF66uNQrncStDiHPQgwvgttA","digest":"69Hzw4IAB6CZ7HWuJQqfgOTvR4NpVRwKhTuuhNrXJCo","integrationUrls":["%s"]}`, validFinalizeServer.URL)

	validServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if !strings.Contains(r.URL.Path, endpointPath) {
			t.Errorf("Expected to hit path containing %s, instead got %s", endpointPath, r.URL.Path)
		}
		if r.URL.RawQuery != endpointQuery {
			t.Errorf("Expected endpoint %s, instead got %s", endpointQuery, r.URL.RawQuery)
		}

		body, _ := io.ReadAll(r.Body)
		bodyS, _ := url.QueryUnescape(string(body))
		if !strings.Contains(bodyS, dropzoneName) {
			t.Errorf("Expected dropzone data to contain name %s, but does not. Body: %s", dropzoneName, bodyS)
		}

		if !strings.Contains(bodyS, dropzoneEmail) {
			t.Errorf("Expected dropzone data to contain email %s, but does not. Body: %s", dropzoneEmail, bodyS)
		}

		if !strings.Contains(bodyS, samplePackageCode) {
			t.Errorf("Expected dropzone data to contain samplePackageCode %s, but does not. Body: %s", samplePackageCode, bodyS)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(validResponse))
	}))

	u := getUploader(validServer.URL)
	p := getPackage(u)

	// test that we don't get errors on submitting a valid bunch of data
	if err := p.SubmitHostedDropzone(dropzoneName, dropzoneEmail); err != nil {
		t.Errorf("Expected successful call to SubmitHostedDropzone, instead got err %s", err)
	}

	invalidResponse := `{"success":"false", "message":"FAILED"}`

	invalidServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(invalidResponse))
	}))

	p = getPackage(getUploader(invalidServer.URL))
	if err := p.SubmitHostedDropzone(dropzoneName, dropzoneEmail); err == nil {
		t.Errorf("Expected an error on dropzone submission due to success: false, instead it succeeded")
	}
}

func TestFinalizeHostedDropzone(t *testing.T) {

	validDZInfo := HostedDropzoneInfo{
		Success: "true",
		Data:    "0--m2RXSf1KHyqg5i9YnRvKr4CBkJci0WnLuW7ZVx0vpf4QdTsBeuxOa6OAvsuZYEVFgTbDAQWoK7dndVSdByKA_NGKcTogLk326jf1wm88Q0T_c7Nyldd3Ibk2mzENL08wsCUbpTfu0lIxnwqOFb27MVFzI9PhOVUI8cslQbOJ40mL_IGF0Z4m8tsQL2-UeUfycS94IrdMxBqp8CSbhoW6CehGwcm7Co-Ubd6C_k3dsdVcOMk1YmE_typNKfb9ZCP9i0prRADirYxAXu6nsJJ2T2QzTusBMMP2f9ReSORuNxvr1XXIsIG-oAIsa116naVE3LRMLwdJCVyb3HymL6bJ0mRFZUEmKg9On7K3zXJgiwpjFX2V7UwRRbJgJvg6Jp1GTx87DSaAxR3auVcBiEJeID5vggGBgVD3X4FPOVnEUFWpdgu10EH9vifrms-rRBRksqclSFb0guiqRi-zVQWMAo0Vp-RbE_SUy7_k2QUUxnhXXdtHPPdMs5n-RGoGs3EqFnQZMaTMxS7Lj6HjRQaQ5n-HYcIDNH6lWUl35jVhtF_eI6StizIEfURCBYbMF66uNQrncStDiHPQgwvgttA",
		Digest:  "69Hzw4IAB6CZ7HWuJQqfgOTvR4NpVRwKhTuuhNrXJCo",
		IntegrationURLs: []string{
			"https://woiugkajwud.execute-api.us-east-1.amazonaws.com/connector/",
		},
	}

	exampleLink := `https://secure-upload.yugabyte.com/receive/?thread=A4Z1-MBLN&packageCode=HvzZGxCLR8h0B3jJDR5WRn20b9Bw42pMUg3wfNSa4Vs#keyCode=rPRrI5PtTrGC_oO_RmOAmJ2ZjU5cCOh6JgyZ9Sp8LYs`

	validResponse := `{"result":"success"}`

	validServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// make sure we're sending data, digest, etc
		body, _ := io.ReadAll(r.Body)
		values, _ := url.ParseQuery(string(body))
		if values.Get("data") != validDZInfo.Data {
			t.Errorf("Expected dropzone data to contain data %s, but does not. Body: %s", validDZInfo.Data, string(body))
		}

		if values.Get("digest") != validDZInfo.Digest {
			t.Errorf("Expected dropzone data to contain digest %s, but does not. Body: %s", validDZInfo.Digest, string(body))
		}

		if values.Get("secureLink") != exampleLink {
			t.Errorf("Expected dropzone data to contain submission link %s, but does not. Body: %s", exampleLink, string(body))
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(validResponse))
	}))
	validServer2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// testing headers in this server
		if r.Header.Get("content-type") != "application/x-www-form-urlencoded" {
			t.Errorf("Expected content-type header x-www-form-urlencoded, instead got %s", r.Header.Get("content-type"))
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(validResponse))
	}))

	// don't use the package URL for this test
	p := getPackage(getUploader("example.com"))
	p.URL = exampleLink

	// test that all URLs get hit
	validDZInfo.IntegrationURLs = []string{validServer.URL, validServer2.URL}

	if err := p.finalizeHostedDropzone(validDZInfo); err != nil {
		t.Errorf("Expected no error from a correctly formed validDZInfo, but instead got %s", err)
	}

}
