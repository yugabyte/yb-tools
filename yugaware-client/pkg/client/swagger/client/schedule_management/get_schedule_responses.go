// Code generated by go-swagger; DO NOT EDIT.

package schedule_management

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
)

// GetScheduleReader is a Reader for the GetSchedule structure.
type GetScheduleReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetScheduleReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetScheduleOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewGetScheduleOK creates a GetScheduleOK with default headers values
func NewGetScheduleOK() *GetScheduleOK {
	return &GetScheduleOK{}
}

/* GetScheduleOK describes a response with status code 200, with default header values.

successful operation
*/
type GetScheduleOK struct {
	Payload *models.Schedule
}

func (o *GetScheduleOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/customers/{cUUID}/schedules/{sUUID}][%d] getScheduleOK  %+v", 200, o.Payload)
}
func (o *GetScheduleOK) GetPayload() *models.Schedule {
	return o.Payload
}

func (o *GetScheduleOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Schedule)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
