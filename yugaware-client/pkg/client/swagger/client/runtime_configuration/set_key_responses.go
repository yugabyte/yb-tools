// Code generated by go-swagger; DO NOT EDIT.

package runtime_configuration

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
)

// SetKeyReader is a Reader for the SetKey structure.
type SetKeyReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *SetKeyReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewSetKeyOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewSetKeyOK creates a SetKeyOK with default headers values
func NewSetKeyOK() *SetKeyOK {
	return &SetKeyOK{}
}

/* SetKeyOK describes a response with status code 200, with default header values.

successful operation
*/
type SetKeyOK struct {
	Payload *models.YBPSuccess
}

func (o *SetKeyOK) Error() string {
	return fmt.Sprintf("[PUT /api/v1/customers/{cUUID}/runtime_config/{scope}/key/{key}][%d] setKeyOK  %+v", 200, o.Payload)
}
func (o *SetKeyOK) GetPayload() *models.YBPSuccess {
	return o.Payload
}

func (o *SetKeyOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.YBPSuccess)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
