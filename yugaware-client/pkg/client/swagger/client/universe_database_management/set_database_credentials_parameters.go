// Code generated by go-swagger; DO NOT EDIT.

package universe_database_management

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

// NewSetDatabaseCredentialsParams creates a new SetDatabaseCredentialsParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewSetDatabaseCredentialsParams() *SetDatabaseCredentialsParams {
	return &SetDatabaseCredentialsParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewSetDatabaseCredentialsParamsWithTimeout creates a new SetDatabaseCredentialsParams object
// with the ability to set a timeout on a request.
func NewSetDatabaseCredentialsParamsWithTimeout(timeout time.Duration) *SetDatabaseCredentialsParams {
	return &SetDatabaseCredentialsParams{
		timeout: timeout,
	}
}

// NewSetDatabaseCredentialsParamsWithContext creates a new SetDatabaseCredentialsParams object
// with the ability to set a context for a request.
func NewSetDatabaseCredentialsParamsWithContext(ctx context.Context) *SetDatabaseCredentialsParams {
	return &SetDatabaseCredentialsParams{
		Context: ctx,
	}
}

// NewSetDatabaseCredentialsParamsWithHTTPClient creates a new SetDatabaseCredentialsParams object
// with the ability to set a custom HTTPClient for a request.
func NewSetDatabaseCredentialsParamsWithHTTPClient(client *http.Client) *SetDatabaseCredentialsParams {
	return &SetDatabaseCredentialsParams{
		HTTPClient: client,
	}
}

/* SetDatabaseCredentialsParams contains all the parameters to send to the API endpoint
   for the set database credentials operation.

   Typically these are written to a http.Request.
*/
type SetDatabaseCredentialsParams struct {

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

// WithDefaults hydrates default values in the set database credentials params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *SetDatabaseCredentialsParams) WithDefaults() *SetDatabaseCredentialsParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the set database credentials params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *SetDatabaseCredentialsParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the set database credentials params
func (o *SetDatabaseCredentialsParams) WithTimeout(timeout time.Duration) *SetDatabaseCredentialsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the set database credentials params
func (o *SetDatabaseCredentialsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the set database credentials params
func (o *SetDatabaseCredentialsParams) WithContext(ctx context.Context) *SetDatabaseCredentialsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the set database credentials params
func (o *SetDatabaseCredentialsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the set database credentials params
func (o *SetDatabaseCredentialsParams) WithHTTPClient(client *http.Client) *SetDatabaseCredentialsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the set database credentials params
func (o *SetDatabaseCredentialsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCUUID adds the cUUID to the set database credentials params
func (o *SetDatabaseCredentialsParams) WithCUUID(cUUID strfmt.UUID) *SetDatabaseCredentialsParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the set database credentials params
func (o *SetDatabaseCredentialsParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WithUniUUID adds the uniUUID to the set database credentials params
func (o *SetDatabaseCredentialsParams) WithUniUUID(uniUUID strfmt.UUID) *SetDatabaseCredentialsParams {
	o.SetUniUUID(uniUUID)
	return o
}

// SetUniUUID adds the uniUuid to the set database credentials params
func (o *SetDatabaseCredentialsParams) SetUniUUID(uniUUID strfmt.UUID) {
	o.UniUUID = uniUUID
}

// WriteToRequest writes these params to a swagger request
func (o *SetDatabaseCredentialsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
