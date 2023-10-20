package sendsafelyuploader

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// testing variables
var (
	testKey    = "testKey"
	testTarget = "DROP_ZONE"

	testOKResponse = `{"value":"OK"}`

	testInvalidResponse = `invalid content`
)

func getUploader(URL string) *Uploader {
	return CreateUploader(URL, testKey, testTarget)
}

func TestSendRequest(t *testing.T) {

	testPath := "/testPath"
	testMethod := http.MethodPut
	testBody := []byte("{type: test}")

	// test OK response
	serverOK := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != testPath {
			t.Errorf("Expected to request '%s', got: %s", testPath, r.URL.Path)
		}
		if r.Header.Get("ss-api-key") != testKey {
			t.Errorf("Expected ss-api-key header: %s, got: %s", testKey, r.Header.Get("ss-api-key"))

		}
		if r.Header.Get("ss-request-api") != testTarget {
			t.Errorf("Expected ss-request-api header: %s, got: %s", testTarget, r.Header.Get("ss-api-key"))
		}

		sentBody, _ := io.ReadAll(r.Body)

		if string(sentBody) != string(testBody) {
			t.Errorf("Expected sent message: %s, got: %s", string(testBody), string(sentBody))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testOKResponse))

	}))
	defer serverOK.Close()

	uploader := CreateUploader(serverOK.URL, testKey, testTarget)

	resp, err := uploader.sendRequest(testMethod, testPath, testBody)

	if err != nil {
		t.Errorf("Expected no error, instead got %s", err)
	}

	if string(resp) != testOKResponse {
		t.Errorf("Expected to get an OK response, instead got %s", string(resp))
	}

	// test Invalid response

	serverInvalid := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(testInvalidResponse))

	}))
	defer serverInvalid.Close()

	uploader = CreateUploader(serverInvalid.URL, testKey, testTarget)

	resp, err = uploader.sendRequest(testMethod, testPath, testBody)

	if err == nil {
		t.Errorf("Expected an error due to invalid HTTP response, instead got %s", err)
	}

	if string(resp) != "" {
		t.Errorf("Expected response '%s', instead got '%s'", testInvalidResponse, string(resp))
	}

}
