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

// NewListOfBackupsParams creates a new ListOfBackupsParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListOfBackupsParams() *ListOfBackupsParams {
	return &ListOfBackupsParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListOfBackupsParamsWithTimeout creates a new ListOfBackupsParams object
// with the ability to set a timeout on a request.
func NewListOfBackupsParamsWithTimeout(timeout time.Duration) *ListOfBackupsParams {
	return &ListOfBackupsParams{
		timeout: timeout,
	}
}

// NewListOfBackupsParamsWithContext creates a new ListOfBackupsParams object
// with the ability to set a context for a request.
func NewListOfBackupsParamsWithContext(ctx context.Context) *ListOfBackupsParams {
	return &ListOfBackupsParams{
		Context: ctx,
	}
}

// NewListOfBackupsParamsWithHTTPClient creates a new ListOfBackupsParams object
// with the ability to set a custom HTTPClient for a request.
func NewListOfBackupsParamsWithHTTPClient(client *http.Client) *ListOfBackupsParams {
	return &ListOfBackupsParams{
		HTTPClient: client,
	}
}

/* ListOfBackupsParams contains all the parameters to send to the API endpoint
   for the list of backups operation.

   Typically these are written to a http.Request.
*/
type ListOfBackupsParams struct {

	// CUUID.
	//
	// Format: uuid
	CUUID strfmt.UUID

	// UniUUID.
	//
	// Format: uuid
	UniUUID strfmt.UUID

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list of backups params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListOfBackupsParams) WithDefaults() *ListOfBackupsParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list of backups params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListOfBackupsParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list of backups params
func (o *ListOfBackupsParams) WithTimeout(timeout time.Duration) *ListOfBackupsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list of backups params
func (o *ListOfBackupsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list of backups params
func (o *ListOfBackupsParams) WithContext(ctx context.Context) *ListOfBackupsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list of backups params
func (o *ListOfBackupsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list of backups params
func (o *ListOfBackupsParams) WithHTTPClient(client *http.Client) *ListOfBackupsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list of backups params
func (o *ListOfBackupsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCUUID adds the cUUID to the list of backups params
func (o *ListOfBackupsParams) WithCUUID(cUUID strfmt.UUID) *ListOfBackupsParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the list of backups params
func (o *ListOfBackupsParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WithUniUUID adds the uniUUID to the list of backups params
func (o *ListOfBackupsParams) WithUniUUID(uniUUID strfmt.UUID) *ListOfBackupsParams {
	o.SetUniUUID(uniUUID)
	return o
}

// SetUniUUID adds the uniUuid to the list of backups params
func (o *ListOfBackupsParams) SetUniUUID(uniUUID strfmt.UUID) {
	o.UniUUID = uniUUID
}

// WriteToRequest writes these params to a swagger request
func (o *ListOfBackupsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param cUUID
	if err := r.SetPathParam("cUUID", o.CUUID.String()); err != nil {
		return err
	}

	// path param uniUUID
	if err := r.SetPathParam("uniUUID", o.UniUUID.String()); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
