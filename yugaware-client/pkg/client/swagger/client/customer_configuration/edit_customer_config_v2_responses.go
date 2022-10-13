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

// EditCustomerConfigV2Reader is a Reader for the EditCustomerConfigV2 structure.
type EditCustomerConfigV2Reader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *EditCustomerConfigV2Reader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewEditCustomerConfigV2OK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewEditCustomerConfigV2OK creates a EditCustomerConfigV2OK with default headers values
func NewEditCustomerConfigV2OK() *EditCustomerConfigV2OK {
	return &EditCustomerConfigV2OK{}
}

/* EditCustomerConfigV2OK describes a response with status code 200, with default header values.

successful operation
*/
type EditCustomerConfigV2OK struct {
	Payload *models.CustomerConfig
}

func (o *EditCustomerConfigV2OK) Error() string {
	return fmt.Sprintf("[PUT /api/v1/customers/{cUUID}/configs/{configUUID}/edit][%d] editCustomerConfigV2OK  %+v", 200, o.Payload)
}
func (o *EditCustomerConfigV2OK) GetPayload() *models.CustomerConfig {
	return o.Payload
}

func (o *EditCustomerConfigV2OK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.CustomerConfig)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}