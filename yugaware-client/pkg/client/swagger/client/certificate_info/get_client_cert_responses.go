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

// GetClientCertReader is a Reader for the GetClientCert structure.
type GetClientCertReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetClientCertReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetClientCertOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewGetClientCertOK creates a GetClientCertOK with default headers values
func NewGetClientCertOK() *GetClientCertOK {
	return &GetClientCertOK{}
}

/* GetClientCertOK describes a response with status code 200, with default header values.

successful operation
*/
type GetClientCertOK struct {
	Payload *models.CertificateDetails
}

func (o *GetClientCertOK) Error() string {
	return fmt.Sprintf("[POST /api/v1/customers/{cUUID}/certificates/{rUUID}][%d] getClientCertOK  %+v", 200, o.Payload)
}
func (o *GetClientCertOK) GetPayload() *models.CertificateDetails {
	return o.Payload
}

func (o *GetClientCertOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.CertificateDetails)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
