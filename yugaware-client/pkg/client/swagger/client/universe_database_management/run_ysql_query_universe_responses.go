// Code generated by go-swagger; DO NOT EDIT.

package universe_database_management

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// RunYsqlQueryUniverseReader is a Reader for the RunYsqlQueryUniverse structure.
type RunYsqlQueryUniverseReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *RunYsqlQueryUniverseReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewRunYsqlQueryUniverseOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewRunYsqlQueryUniverseOK creates a RunYsqlQueryUniverseOK with default headers values
func NewRunYsqlQueryUniverseOK() *RunYsqlQueryUniverseOK {
	return &RunYsqlQueryUniverseOK{}
}

/* RunYsqlQueryUniverseOK describes a response with status code 200, with default header values.

successful operation
*/
type RunYsqlQueryUniverseOK struct {
	Payload interface{}
}

func (o *RunYsqlQueryUniverseOK) Error() string {
	return fmt.Sprintf("[POST /api/v1/customers/{cUUID}/universes/{uniUUID}/run_query][%d] runYsqlQueryUniverseOK  %+v", 200, o.Payload)
}
func (o *RunYsqlQueryUniverseOK) GetPayload() interface{} {
	return o.Payload
}

func (o *RunYsqlQueryUniverseOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
