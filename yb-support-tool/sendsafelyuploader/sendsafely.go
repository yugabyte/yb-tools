package sendsafelyuploader

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Uploader struct {
	URL              string
	APIKey           string
	RequestAPITarget string
	ClientSecret     string
	Client           *http.Client
}

func CreateUploader(SSUrl, SSAPIKey, SSRequestTarget string) *Uploader {
	u := Uploader{URL: SSUrl,
		APIKey:           SSAPIKey,
		RequestAPITarget: SSRequestTarget,
		ClientSecret:     createClientSecret(),
		Client:           &http.Client{},
	}
	return &u
}

type uploadReqOption func(*http.Request) error

func withReqJSONHeader() uploadReqOption {
	return func(r *http.Request) error {
		r.Header.Set("content-type", "application/json;charset=utf-8")
		return nil
	}
}

func withReqFormHeader() uploadReqOption {
	return func(r *http.Request) error {
		r.Header.Set("content-type", "application/x-www-form-urlencoded")
		return nil
	}
}

// uses uploader credentials to send the provided body to the provided enpoint
// returns the response body as an array of bytes and any errors
func (u *Uploader) sendRequest(method, endpoint string, body []byte, reqOptions ...uploadReqOption) ([]byte, error) {
	req, err := http.NewRequest(method, u.URL+endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("ss-api-key", u.APIKey)
	req.Header.Set("ss-request-api", u.RequestAPITarget)

	for _, option := range reqOptions {
		if err := option(req); err != nil {
			return nil, err
		}
	}

	resp, err := u.Client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("Invalid HTTP Status Code: %d; Unable to read response body", resp.StatusCode)
		}
		return nil, fmt.Errorf("Invalid HTTP status response; HTTP Error code %d; request body %s", resp.StatusCode, string(respBody))
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)

}
