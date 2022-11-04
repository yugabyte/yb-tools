// Code generated by go-swagger; DO NOT EDIT.

package customer_configuration

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
)

// EditCustomerConfigReader is a Reader for the EditCustomerConfig structure.
type EditCustomerConfigReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *EditCustomerConfigReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewEditCustomerConfigOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewEditCustomerConfigOK creates a EditCustomerConfigOK with default headers values
func NewEditCustomerConfigOK() *EditCustomerConfigOK {
	return &EditCustomerConfigOK{}
}

/* EditCustomerConfigOK describes a response with status code 200, with default header values.

successful operation
*/
type EditCustomerConfigOK struct {
	Payload *models.CustomerConfig
}

func (o *EditCustomerConfigOK) Error() string {
	return fmt.Sprintf("[PUT /api/v1/customers/{cUUID}/configs/{configUUID}][%d] editCustomerConfigOK  %+v", 200, o.Payload)
}
func (o *EditCustomerConfigOK) GetPayload() *models.CustomerConfig {
	return o.Payload
}

func (o *EditCustomerConfigOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.CustomerConfig)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
