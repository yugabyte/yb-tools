// Code generated by go-swagger; DO NOT EDIT.

package universe_information

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
)

// GetUniverseCostReader is a Reader for the GetUniverseCost structure.
type GetUniverseCostReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetUniverseCostReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetUniverseCostOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewGetUniverseCostOK creates a GetUniverseCostOK with default headers values
func NewGetUniverseCostOK() *GetUniverseCostOK {
	return &GetUniverseCostOK{}
}

/* GetUniverseCostOK describes a response with status code 200, with default header values.

successful operation
*/
type GetUniverseCostOK struct {
	Payload *models.UniverseResourceDetails
}

func (o *GetUniverseCostOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/customers/{cUUID}/universes/{uniUUID}/cost][%d] getUniverseCostOK  %+v", 200, o.Payload)
}
func (o *GetUniverseCostOK) GetPayload() *models.UniverseResourceDetails {
	return o.Payload
}

func (o *GetUniverseCostOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.UniverseResourceDetails)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
