// Code generated by go-swagger; DO NOT EDIT.

package universe_node_metadata_metamaster

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

// NewGetRedisServerAddressesParams creates a new GetRedisServerAddressesParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewGetRedisServerAddressesParams() *GetRedisServerAddressesParams {
	return &GetRedisServerAddressesParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewGetRedisServerAddressesParamsWithTimeout creates a new GetRedisServerAddressesParams object
// with the ability to set a timeout on a request.
func NewGetRedisServerAddressesParamsWithTimeout(timeout time.Duration) *GetRedisServerAddressesParams {
	return &GetRedisServerAddressesParams{
		timeout: timeout,
	}
}

// NewGetRedisServerAddressesParamsWithContext creates a new GetRedisServerAddressesParams object
// with the ability to set a context for a request.
func NewGetRedisServerAddressesParamsWithContext(ctx context.Context) *GetRedisServerAddressesParams {
	return &GetRedisServerAddressesParams{
		Context: ctx,
	}
}

// NewGetRedisServerAddressesParamsWithHTTPClient creates a new GetRedisServerAddressesParams object
// with the ability to set a custom HTTPClient for a request.
func NewGetRedisServerAddressesParamsWithHTTPClient(client *http.Client) *GetRedisServerAddressesParams {
	return &GetRedisServerAddressesParams{
		HTTPClient: client,
	}
}

/* GetRedisServerAddressesParams contains all the parameters to send to the API endpoint
   for the get redis server addresses operation.

   Typically these are written to a http.Request.
*/
type GetRedisServerAddressesParams struct {

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

// WithDefaults hydrates default values in the get redis server addresses params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetRedisServerAddressesParams) WithDefaults() *GetRedisServerAddressesParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the get redis server addresses params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetRedisServerAddressesParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the get redis server addresses params
func (o *GetRedisServerAddressesParams) WithTimeout(timeout time.Duration) *GetRedisServerAddressesParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get redis server addresses params
func (o *GetRedisServerAddressesParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get redis server addresses params
func (o *GetRedisServerAddressesParams) WithContext(ctx context.Context) *GetRedisServerAddressesParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get redis server addresses params
func (o *GetRedisServerAddressesParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get redis server addresses params
func (o *GetRedisServerAddressesParams) WithHTTPClient(client *http.Client) *GetRedisServerAddressesParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get redis server addresses params
func (o *GetRedisServerAddressesParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCUUID adds the cUUID to the get redis server addresses params
func (o *GetRedisServerAddressesParams) WithCUUID(cUUID strfmt.UUID) *GetRedisServerAddressesParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the get redis server addresses params
func (o *GetRedisServerAddressesParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WithUniUUID adds the uniUUID to the get redis server addresses params
func (o *GetRedisServerAddressesParams) WithUniUUID(uniUUID strfmt.UUID) *GetRedisServerAddressesParams {
	o.SetUniUUID(uniUUID)
	return o
}

// SetUniUUID adds the uniUuid to the get redis server addresses params
func (o *GetRedisServerAddressesParams) SetUniUUID(uniUUID strfmt.UUID) {
	o.UniUUID = uniUUID
}

// WriteToRequest writes these params to a swagger request
func (o *GetRedisServerAddressesParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
