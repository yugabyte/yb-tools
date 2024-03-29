// Code generated by go-swagger; DO NOT EDIT.

package backups

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
)

// ListOfBackupsReader is a Reader for the ListOfBackups structure.
type ListOfBackupsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListOfBackupsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListOfBackupsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 500:
		result := NewListOfBackupsInternalServerError()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewListOfBackupsOK creates a ListOfBackupsOK with default headers values
func NewListOfBackupsOK() *ListOfBackupsOK {
	return &ListOfBackupsOK{}
}

/* ListOfBackupsOK describes a response with status code 200, with default header values.

successful operation
*/
type ListOfBackupsOK struct {
	Payload []*models.Backup
}

func (o *ListOfBackupsOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/customers/{cUUID}/universes/{uniUUID}/backups][%d] listOfBackupsOK  %+v", 200, o.Payload)
}
func (o *ListOfBackupsOK) GetPayload() []*models.Backup {
	return o.Payload
}

func (o *ListOfBackupsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListOfBackupsInternalServerError creates a ListOfBackupsInternalServerError with default headers values
func NewListOfBackupsInternalServerError() *ListOfBackupsInternalServerError {
	return &ListOfBackupsInternalServerError{}
}

/* ListOfBackupsInternalServerError describes a response with status code 500, with default header values.

If there was a server or database issue when listing the backups
*/
type ListOfBackupsInternalServerError struct {
	Payload *models.YBPError
}

func (o *ListOfBackupsInternalServerError) Error() string {
	return fmt.Sprintf("[GET /api/v1/customers/{cUUID}/universes/{uniUUID}/backups][%d] listOfBackupsInternalServerError  %+v", 500, o.Payload)
}
func (o *ListOfBackupsInternalServerError) GetPayload() *models.YBPError {
	return o.Payload
}

func (o *ListOfBackupsInternalServerError) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.YBPError)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
