// Code generated by go-swagger; DO NOT EDIT.

package logging_config

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
)

// SetLoggingSettingsReader is a Reader for the SetLoggingSettings structure.
type SetLoggingSettingsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *SetLoggingSettingsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewSetLoggingSettingsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewSetLoggingSettingsOK creates a SetLoggingSettingsOK with default headers values
func NewSetLoggingSettingsOK() *SetLoggingSettingsOK {
	return &SetLoggingSettingsOK{}
}

/* SetLoggingSettingsOK describes a response with status code 200, with default header values.

successful operation
*/
type SetLoggingSettingsOK struct {
	Payload *models.PlatformLoggingConfig
}

func (o *SetLoggingSettingsOK) Error() string {
	return fmt.Sprintf("[POST /api/v1/logging_config][%d] setLoggingSettingsOK  %+v", 200, o.Payload)
}
func (o *SetLoggingSettingsOK) GetPayload() *models.PlatformLoggingConfig {
	return o.Payload
}

func (o *SetLoggingSettingsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.PlatformLoggingConfig)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
