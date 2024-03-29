// Code generated by go-swagger; DO NOT EDIT.

package release_management

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
)

// UpdateReleaseReader is a Reader for the UpdateRelease structure.
type UpdateReleaseReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *UpdateReleaseReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewUpdateReleaseOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewUpdateReleaseOK creates a UpdateReleaseOK with default headers values
func NewUpdateReleaseOK() *UpdateReleaseOK {
	return &UpdateReleaseOK{}
}

/* UpdateReleaseOK describes a response with status code 200, with default header values.

successful operation
*/
type UpdateReleaseOK struct {
	Payload *models.ReleaseMetadata
}

func (o *UpdateReleaseOK) Error() string {
	return fmt.Sprintf("[PUT /api/v1/customers/{cUUID}/releases/{name}][%d] updateReleaseOK  %+v", 200, o.Payload)
}
func (o *UpdateReleaseOK) GetPayload() *models.ReleaseMetadata {
	return o.Payload
}

func (o *UpdateReleaseOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ReleaseMetadata)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
