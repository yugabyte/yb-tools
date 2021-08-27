// Code generated by go-swagger; DO NOT EDIT.

package backup_schedule_management

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"net/http"
	"time"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	cr "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

// NewDeleteBackupScheduleParams creates a new DeleteBackupScheduleParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewDeleteBackupScheduleParams() *DeleteBackupScheduleParams {
	return &DeleteBackupScheduleParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewDeleteBackupScheduleParamsWithTimeout creates a new DeleteBackupScheduleParams object
// with the ability to set a timeout on a request.
func NewDeleteBackupScheduleParamsWithTimeout(timeout time.Duration) *DeleteBackupScheduleParams {
	return &DeleteBackupScheduleParams{
		timeout: timeout,
	}
}

// NewDeleteBackupScheduleParamsWithContext creates a new DeleteBackupScheduleParams object
// with the ability to set a context for a request.
func NewDeleteBackupScheduleParamsWithContext(ctx context.Context) *DeleteBackupScheduleParams {
	return &DeleteBackupScheduleParams{
		Context: ctx,
	}
}

// NewDeleteBackupScheduleParamsWithHTTPClient creates a new DeleteBackupScheduleParams object
// with the ability to set a custom HTTPClient for a request.
func NewDeleteBackupScheduleParamsWithHTTPClient(client *http.Client) *DeleteBackupScheduleParams {
	return &DeleteBackupScheduleParams{
		HTTPClient: client,
	}
}

/* DeleteBackupScheduleParams contains all the parameters to send to the API endpoint
   for the delete backup schedule operation.

   Typically these are written to a http.Request.
*/
type DeleteBackupScheduleParams struct {

	// CUUID.
	//
	// Format: uuid
	CUUID strfmt.UUID

	// SUUID.
	//
	// Format: uuid
	SUUID strfmt.UUID

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the delete backup schedule params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteBackupScheduleParams) WithDefaults() *DeleteBackupScheduleParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the delete backup schedule params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteBackupScheduleParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the delete backup schedule params
func (o *DeleteBackupScheduleParams) WithTimeout(timeout time.Duration) *DeleteBackupScheduleParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the delete backup schedule params
func (o *DeleteBackupScheduleParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the delete backup schedule params
func (o *DeleteBackupScheduleParams) WithContext(ctx context.Context) *DeleteBackupScheduleParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the delete backup schedule params
func (o *DeleteBackupScheduleParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the delete backup schedule params
func (o *DeleteBackupScheduleParams) WithHTTPClient(client *http.Client) *DeleteBackupScheduleParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the delete backup schedule params
func (o *DeleteBackupScheduleParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCUUID adds the cUUID to the delete backup schedule params
func (o *DeleteBackupScheduleParams) WithCUUID(cUUID strfmt.UUID) *DeleteBackupScheduleParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the delete backup schedule params
func (o *DeleteBackupScheduleParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WithSUUID adds the sUUID to the delete backup schedule params
func (o *DeleteBackupScheduleParams) WithSUUID(sUUID strfmt.UUID) *DeleteBackupScheduleParams {
	o.SetSUUID(sUUID)
	return o
}

// SetSUUID adds the sUuid to the delete backup schedule params
func (o *DeleteBackupScheduleParams) SetSUUID(sUUID strfmt.UUID) {
	o.SUUID = sUUID
}

// WriteToRequest writes these params to a swagger request
func (o *DeleteBackupScheduleParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param cUUID
	if err := r.SetPathParam("cUUID", o.CUUID.String()); err != nil {
		return err
	}

	// path param sUUID
	if err := r.SetPathParam("sUUID", o.SUUID.String()); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}