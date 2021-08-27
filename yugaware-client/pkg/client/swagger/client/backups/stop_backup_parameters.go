// Code generated by go-swagger; DO NOT EDIT.

package backups

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

// NewStopBackupParams creates a new StopBackupParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewStopBackupParams() *StopBackupParams {
	return &StopBackupParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewStopBackupParamsWithTimeout creates a new StopBackupParams object
// with the ability to set a timeout on a request.
func NewStopBackupParamsWithTimeout(timeout time.Duration) *StopBackupParams {
	return &StopBackupParams{
		timeout: timeout,
	}
}

// NewStopBackupParamsWithContext creates a new StopBackupParams object
// with the ability to set a context for a request.
func NewStopBackupParamsWithContext(ctx context.Context) *StopBackupParams {
	return &StopBackupParams{
		Context: ctx,
	}
}

// NewStopBackupParamsWithHTTPClient creates a new StopBackupParams object
// with the ability to set a custom HTTPClient for a request.
func NewStopBackupParamsWithHTTPClient(client *http.Client) *StopBackupParams {
	return &StopBackupParams{
		HTTPClient: client,
	}
}

/* StopBackupParams contains all the parameters to send to the API endpoint
   for the stop backup operation.

   Typically these are written to a http.Request.
*/
type StopBackupParams struct {

	// BackupUUID.
	//
	// Format: uuid
	BackupUUID strfmt.UUID

	// CUUID.
	//
	// Format: uuid
	CUUID strfmt.UUID

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the stop backup params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *StopBackupParams) WithDefaults() *StopBackupParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the stop backup params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *StopBackupParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the stop backup params
func (o *StopBackupParams) WithTimeout(timeout time.Duration) *StopBackupParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the stop backup params
func (o *StopBackupParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the stop backup params
func (o *StopBackupParams) WithContext(ctx context.Context) *StopBackupParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the stop backup params
func (o *StopBackupParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the stop backup params
func (o *StopBackupParams) WithHTTPClient(client *http.Client) *StopBackupParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the stop backup params
func (o *StopBackupParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithBackupUUID adds the backupUUID to the stop backup params
func (o *StopBackupParams) WithBackupUUID(backupUUID strfmt.UUID) *StopBackupParams {
	o.SetBackupUUID(backupUUID)
	return o
}

// SetBackupUUID adds the backupUuid to the stop backup params
func (o *StopBackupParams) SetBackupUUID(backupUUID strfmt.UUID) {
	o.BackupUUID = backupUUID
}

// WithCUUID adds the cUUID to the stop backup params
func (o *StopBackupParams) WithCUUID(cUUID strfmt.UUID) *StopBackupParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the stop backup params
func (o *StopBackupParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WriteToRequest writes these params to a swagger request
func (o *StopBackupParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param backupUUID
	if err := r.SetPathParam("backupUUID", o.BackupUUID.String()); err != nil {
		return err
	}

	// path param cUUID
	if err := r.SetPathParam("cUUID", o.CUUID.String()); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}