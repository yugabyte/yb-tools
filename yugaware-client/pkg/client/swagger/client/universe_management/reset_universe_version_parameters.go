// Code generated by go-swagger; DO NOT EDIT.

package universe_management

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

// NewResetUniverseVersionParams creates a new ResetUniverseVersionParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewResetUniverseVersionParams() *ResetUniverseVersionParams {
	return &ResetUniverseVersionParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewResetUniverseVersionParamsWithTimeout creates a new ResetUniverseVersionParams object
// with the ability to set a timeout on a request.
func NewResetUniverseVersionParamsWithTimeout(timeout time.Duration) *ResetUniverseVersionParams {
	return &ResetUniverseVersionParams{
		timeout: timeout,
	}
}

// NewResetUniverseVersionParamsWithContext creates a new ResetUniverseVersionParams object
// with the ability to set a context for a request.
func NewResetUniverseVersionParamsWithContext(ctx context.Context) *ResetUniverseVersionParams {
	return &ResetUniverseVersionParams{
		Context: ctx,
	}
}

// NewResetUniverseVersionParamsWithHTTPClient creates a new ResetUniverseVersionParams object
// with the ability to set a custom HTTPClient for a request.
func NewResetUniverseVersionParamsWithHTTPClient(client *http.Client) *ResetUniverseVersionParams {
	return &ResetUniverseVersionParams{
		HTTPClient: client,
	}
}

/* ResetUniverseVersionParams contains all the parameters to send to the API endpoint
   for the reset universe version operation.

   Typically these are written to a http.Request.
*/
type ResetUniverseVersionParams struct {

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

// WithDefaults hydrates default values in the reset universe version params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ResetUniverseVersionParams) WithDefaults() *ResetUniverseVersionParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the reset universe version params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ResetUniverseVersionParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the reset universe version params
func (o *ResetUniverseVersionParams) WithTimeout(timeout time.Duration) *ResetUniverseVersionParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the reset universe version params
func (o *ResetUniverseVersionParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the reset universe version params
func (o *ResetUniverseVersionParams) WithContext(ctx context.Context) *ResetUniverseVersionParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the reset universe version params
func (o *ResetUniverseVersionParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the reset universe version params
func (o *ResetUniverseVersionParams) WithHTTPClient(client *http.Client) *ResetUniverseVersionParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the reset universe version params
func (o *ResetUniverseVersionParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCUUID adds the cUUID to the reset universe version params
func (o *ResetUniverseVersionParams) WithCUUID(cUUID strfmt.UUID) *ResetUniverseVersionParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the reset universe version params
func (o *ResetUniverseVersionParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WithUniUUID adds the uniUUID to the reset universe version params
func (o *ResetUniverseVersionParams) WithUniUUID(uniUUID strfmt.UUID) *ResetUniverseVersionParams {
	o.SetUniUUID(uniUUID)
	return o
}

// SetUniUUID adds the uniUuid to the reset universe version params
func (o *ResetUniverseVersionParams) SetUniUUID(uniUUID strfmt.UUID) {
	o.UniUUID = uniUUID
}

// WriteToRequest writes these params to a swagger request
func (o *ResetUniverseVersionParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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