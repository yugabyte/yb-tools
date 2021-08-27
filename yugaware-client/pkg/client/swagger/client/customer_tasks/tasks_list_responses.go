// Code generated by go-swagger; DO NOT EDIT.

package customer_tasks

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
)

// TasksListReader is a Reader for the TasksList structure.
type TasksListReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *TasksListReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewTasksListOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewTasksListOK creates a TasksListOK with default headers values
func NewTasksListOK() *TasksListOK {
	return &TasksListOK{}
}

/* TasksListOK describes a response with status code 200, with default header values.

successful operation
*/
type TasksListOK struct {
	Payload []*models.CustomerTaskData
}

func (o *TasksListOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/customers/{cUUID}/tasks_list][%d] tasksListOK  %+v", 200, o.Payload)
}
func (o *TasksListOK) GetPayload() []*models.CustomerTaskData {
	return o.Payload
}

func (o *TasksListOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}