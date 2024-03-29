// Code generated by go-swagger; DO NOT EDIT.

package certificate_info

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
)

// EditCertificateReader is a Reader for the EditCertificate structure.
type EditCertificateReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *EditCertificateReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewEditCertificateOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewEditCertificateOK creates a EditCertificateOK with default headers values
func NewEditCertificateOK() *EditCertificateOK {
	return &EditCertificateOK{}
}

/* EditCertificateOK describes a response with status code 200, with default header values.

successful operation
*/
type EditCertificateOK struct {
	Payload *models.YBPSuccess
}

func (o *EditCertificateOK) Error() string {
	return fmt.Sprintf("[POST /api/v1/customers/{cUUID}/certificates/{rUUID}/edit][%d] editCertificateOK  %+v", 200, o.Payload)
}
func (o *EditCertificateOK) GetPayload() *models.YBPSuccess {
	return o.Payload
}

func (o *EditCertificateOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.YBPSuccess)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
