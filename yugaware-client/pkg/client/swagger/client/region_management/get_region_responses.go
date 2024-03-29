// Code generated by go-swagger; DO NOT EDIT.

package region_management

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
)

// GetRegionReader is a Reader for the GetRegion structure.
type GetRegionReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetRegionReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetRegionOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 500:
		result := NewGetRegionInternalServerError()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewGetRegionOK creates a GetRegionOK with default headers values
func NewGetRegionOK() *GetRegionOK {
	return &GetRegionOK{}
}

/* GetRegionOK describes a response with status code 200, with default header values.

successful operation
*/
type GetRegionOK struct {
	Payload []*models.Region
}

func (o *GetRegionOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/customers/{cUUID}/providers/{pUUID}/regions][%d] getRegionOK  %+v", 200, o.Payload)
}
func (o *GetRegionOK) GetPayload() []*models.Region {
	return o.Payload
}

func (o *GetRegionOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetRegionInternalServerError creates a GetRegionInternalServerError with default headers values
func NewGetRegionInternalServerError() *GetRegionInternalServerError {
	return &GetRegionInternalServerError{}
}

/* GetRegionInternalServerError describes a response with status code 500, with default header values.

If there was a server or database issue when listing the regions
*/
type GetRegionInternalServerError struct {
	Payload *models.YBPError
}

func (o *GetRegionInternalServerError) Error() string {
	return fmt.Sprintf("[GET /api/v1/customers/{cUUID}/providers/{pUUID}/regions][%d] getRegionInternalServerError  %+v", 500, o.Payload)
}
func (o *GetRegionInternalServerError) GetPayload() *models.YBPError {
	return o.Payload
}

func (o *GetRegionInternalServerError) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.YBPError)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
