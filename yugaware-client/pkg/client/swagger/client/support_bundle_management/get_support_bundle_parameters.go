// Code generated by go-swagger; DO NOT EDIT.

package support_bundle_management

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

// NewGetSupportBundleParams creates a new GetSupportBundleParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewGetSupportBundleParams() *GetSupportBundleParams {
	return &GetSupportBundleParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewGetSupportBundleParamsWithTimeout creates a new GetSupportBundleParams object
// with the ability to set a timeout on a request.
func NewGetSupportBundleParamsWithTimeout(timeout time.Duration) *GetSupportBundleParams {
	return &GetSupportBundleParams{
		timeout: timeout,
	}
}

// NewGetSupportBundleParamsWithContext creates a new GetSupportBundleParams object
// with the ability to set a context for a request.
func NewGetSupportBundleParamsWithContext(ctx context.Context) *GetSupportBundleParams {
	return &GetSupportBundleParams{
		Context: ctx,
	}
}

// NewGetSupportBundleParamsWithHTTPClient creates a new GetSupportBundleParams object
// with the ability to set a custom HTTPClient for a request.
func NewGetSupportBundleParamsWithHTTPClient(client *http.Client) *GetSupportBundleParams {
	return &GetSupportBundleParams{
		HTTPClient: client,
	}
}

/* GetSupportBundleParams contains all the parameters to send to the API endpoint
   for the get support bundle operation.

   Typically these are written to a http.Request.
*/
type GetSupportBundleParams struct {

	// CUUID.
	//
	// Format: uuid
	CUUID strfmt.UUID

	// SbUUID.
	//
	// Format: uuid
	SbUUID strfmt.UUID

	// UniUUID.
	//
	// Format: uuid
	UniUUID strfmt.UUID

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the get support bundle params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetSupportBundleParams) WithDefaults() *GetSupportBundleParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the get support bundle params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetSupportBundleParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the get support bundle params
func (o *GetSupportBundleParams) WithTimeout(timeout time.Duration) *GetSupportBundleParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get support bundle params
func (o *GetSupportBundleParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get support bundle params
func (o *GetSupportBundleParams) WithContext(ctx context.Context) *GetSupportBundleParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get support bundle params
func (o *GetSupportBundleParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get support bundle params
func (o *GetSupportBundleParams) WithHTTPClient(client *http.Client) *GetSupportBundleParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get support bundle params
func (o *GetSupportBundleParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCUUID adds the cUUID to the get support bundle params
func (o *GetSupportBundleParams) WithCUUID(cUUID strfmt.UUID) *GetSupportBundleParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the get support bundle params
func (o *GetSupportBundleParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WithSbUUID adds the sbUUID to the get support bundle params
func (o *GetSupportBundleParams) WithSbUUID(sbUUID strfmt.UUID) *GetSupportBundleParams {
	o.SetSbUUID(sbUUID)
	return o
}

// SetSbUUID adds the sbUuid to the get support bundle params
func (o *GetSupportBundleParams) SetSbUUID(sbUUID strfmt.UUID) {
	o.SbUUID = sbUUID
}

// WithUniUUID adds the uniUUID to the get support bundle params
func (o *GetSupportBundleParams) WithUniUUID(uniUUID strfmt.UUID) *GetSupportBundleParams {
	o.SetUniUUID(uniUUID)
	return o
}

// SetUniUUID adds the uniUuid to the get support bundle params
func (o *GetSupportBundleParams) SetUniUUID(uniUUID strfmt.UUID) {
	o.UniUUID = uniUUID
}

// WriteToRequest writes these params to a swagger request
func (o *GetSupportBundleParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param cUUID
	if err := r.SetPathParam("cUUID", o.CUUID.String()); err != nil {
		return err
	}

	// path param sbUUID
	if err := r.SetPathParam("sbUUID", o.SbUUID.String()); err != nil {
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
