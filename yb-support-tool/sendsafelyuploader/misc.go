package sendsafelyuploader

import "os"

type RequestInfo struct {
	Url                string
	SsApiKey           string
	SsRequestApiTarget string
}

type SSResponse struct {
	Response string `json:"response"`
	Message  string `json:"message"`
}

type FinalizeInfo struct {
	NeedsLink bool `json:"needsLink"`

	SSResponse
}

type HostedDropzoneInfo struct {
	Success         string   `json:"success"`
	Data            string   `json:"data"`
	Digest          string   `json:"digest"`
	IntegrationURLs []string `json:"integrationUrls"`

	SSResponse
}

type File struct {
	Info       *FileInfo // May not need
	Ptr        *os.File
	PartURLs   []string //URL
	UploadInfo *FileUpload
	Status     chan int
}

type FilePart struct {
	Number    int
	Offset    int64 // equal to ChunkSize * Number
	URL       string
	ChunkSize int64
}

type FileUpload struct {
	Name       string `json:"filename"`
	UploadType string `json:"uploadType"`
	Parts      int    `json:"parts"`
	Part       int    `json:"part"`
	Size       int64  `json:"filesize,omitempty,string"`
}

type FileInfo struct {
	ID             string `json:"fileId"`
	Name           string `json:"fileName"`
	Size           int64  `json:"fileSize,omitempty,string"`
	Parts          int    `json:"parts"`
	Uploaded       string `json:"fileUploaded"`
	UploadedStr    string `json:"fileUploadedStr"`
	Version        string `json:"fileVersion"`
	CreatedByEmail string `json:"createdByEmail"`

	SSResponse
}

type UploadUrlInfo struct {
	URLS []UploadURL `json:"uploadUrls"`

	SSResponse
}

type UploadURL struct {
	Part int    `json:"part"`
	URL  string `json:"url"`
}
