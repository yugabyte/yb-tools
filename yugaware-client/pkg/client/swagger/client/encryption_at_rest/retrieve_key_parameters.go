// Code generated by go-swagger; DO NOT EDIT.

package encryption_at_rest

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

// NewRetrieveKeyParams creates a new RetrieveKeyParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewRetrieveKeyParams() *RetrieveKeyParams {
	return &RetrieveKeyParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewRetrieveKeyParamsWithTimeout creates a new RetrieveKeyParams object
// with the ability to set a timeout on a request.
func NewRetrieveKeyParamsWithTimeout(timeout time.Duration) *RetrieveKeyParams {
	return &RetrieveKeyParams{
		timeout: timeout,
	}
}

// NewRetrieveKeyParamsWithContext creates a new RetrieveKeyParams object
// with the ability to set a context for a request.
func NewRetrieveKeyParamsWithContext(ctx context.Context) *RetrieveKeyParams {
	return &RetrieveKeyParams{
		Context: ctx,
	}
}

// NewRetrieveKeyParamsWithHTTPClient creates a new RetrieveKeyParams object
// with the ability to set a custom HTTPClient for a request.
func NewRetrieveKeyParamsWithHTTPClient(client *http.Client) *RetrieveKeyParams {
	return &RetrieveKeyParams{
		HTTPClient: client,
	}
}

/* RetrieveKeyParams contains all the parameters to send to the API endpoint
   for the retrieve key operation.

   Typically these are written to a http.Request.
*/
type RetrieveKeyParams struct {

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

// WithDefaults hydrates default values in the retrieve key params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *RetrieveKeyParams) WithDefaults() *RetrieveKeyParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the retrieve key params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *RetrieveKeyParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the retrieve key params
func (o *RetrieveKeyParams) WithTimeout(timeout time.Duration) *RetrieveKeyParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the retrieve key params
func (o *RetrieveKeyParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the retrieve key params
func (o *RetrieveKeyParams) WithContext(ctx context.Context) *RetrieveKeyParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the retrieve key params
func (o *RetrieveKeyParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the retrieve key params
func (o *RetrieveKeyParams) WithHTTPClient(client *http.Client) *RetrieveKeyParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the retrieve key params
func (o *RetrieveKeyParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCUUID adds the cUUID to the retrieve key params
func (o *RetrieveKeyParams) WithCUUID(cUUID strfmt.UUID) *RetrieveKeyParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the retrieve key params
func (o *RetrieveKeyParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WithUniUUID adds the uniUUID to the retrieve key params
func (o *RetrieveKeyParams) WithUniUUID(uniUUID strfmt.UUID) *RetrieveKeyParams {
	o.SetUniUUID(uniUUID)
	return o
}

// SetUniUUID adds the uniUuid to the retrieve key params
func (o *RetrieveKeyParams) SetUniUUID(uniUUID strfmt.UUID) {
	o.UniUUID = uniUUID
}

// WriteToRequest writes these params to a swagger request
func (o *RetrieveKeyParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
