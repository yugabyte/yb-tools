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

// CreateMultiTableBackupReader is a Reader for the CreateMultiTableBackup structure.
type CreateMultiTableBackupReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *CreateMultiTableBackupReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewCreateMultiTableBackupOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 400:
		result := NewCreateMultiTableBackupBadRequest()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewCreateMultiTableBackupOK creates a CreateMultiTableBackupOK with default headers values
func NewCreateMultiTableBackupOK() *CreateMultiTableBackupOK {
	return &CreateMultiTableBackupOK{}
}

/* CreateMultiTableBackupOK describes a response with status code 200, with default header values.

If requested schedule backup. Changes from upstream: This API call actually returns a task, not a Schedule
*/
type CreateMultiTableBackupOK struct {
	Payload *models.YBPTask
}

func (o *CreateMultiTableBackupOK) Error() string {
	return fmt.Sprintf("[PUT /api/v1/customers/{cUUID}/universes/{uniUUID}/multi_table_backup][%d] createMultiTableBackupOK  %+v", 200, o.Payload)
}
func (o *CreateMultiTableBackupOK) GetPayload() *models.YBPTask {
	return o.Payload
}

func (o *CreateMultiTableBackupOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.YBPTask)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewCreateMultiTableBackupBadRequest creates a CreateMultiTableBackupBadRequest with default headers values
func NewCreateMultiTableBackupBadRequest() *CreateMultiTableBackupBadRequest {
	return &CreateMultiTableBackupBadRequest{}
}

/* CreateMultiTableBackupBadRequest describes a response with status code 400, with default header values.

When request fails validations. Changes from upstream: Return code 400 is not in swagger spec
*/
type CreateMultiTableBackupBadRequest struct {
	Payload *models.YBPError
}

func (o *CreateMultiTableBackupBadRequest) Error() string {
	return fmt.Sprintf("[PUT /api/v1/customers/{cUUID}/universes/{uniUUID}/multi_table_backup][%d] createMultiTableBackupBadRequest  %+v", 400, o.Payload)
}
func (o *CreateMultiTableBackupBadRequest) GetPayload() *models.YBPError {
	return o.Payload
}

func (o *CreateMultiTableBackupBadRequest) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.YBPError)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
