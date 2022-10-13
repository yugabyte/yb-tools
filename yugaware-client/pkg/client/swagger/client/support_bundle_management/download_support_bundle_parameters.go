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

// NewDownloadSupportBundleParams creates a new DownloadSupportBundleParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewDownloadSupportBundleParams() *DownloadSupportBundleParams {
	return &DownloadSupportBundleParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewDownloadSupportBundleParamsWithTimeout creates a new DownloadSupportBundleParams object
// with the ability to set a timeout on a request.
func NewDownloadSupportBundleParamsWithTimeout(timeout time.Duration) *DownloadSupportBundleParams {
	return &DownloadSupportBundleParams{
		timeout: timeout,
	}
}

// NewDownloadSupportBundleParamsWithContext creates a new DownloadSupportBundleParams object
// with the ability to set a context for a request.
func NewDownloadSupportBundleParamsWithContext(ctx context.Context) *DownloadSupportBundleParams {
	return &DownloadSupportBundleParams{
		Context: ctx,
	}
}

// NewDownloadSupportBundleParamsWithHTTPClient creates a new DownloadSupportBundleParams object
// with the ability to set a custom HTTPClient for a request.
func NewDownloadSupportBundleParamsWithHTTPClient(client *http.Client) *DownloadSupportBundleParams {
	return &DownloadSupportBundleParams{
		HTTPClient: client,
	}
}

/* DownloadSupportBundleParams contains all the parameters to send to the API endpoint
   for the download support bundle operation.

   Typically these are written to a http.Request.
*/
type DownloadSupportBundleParams struct {

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

// WithDefaults hydrates default values in the download support bundle params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DownloadSupportBundleParams) WithDefaults() *DownloadSupportBundleParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the download support bundle params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DownloadSupportBundleParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the download support bundle params
func (o *DownloadSupportBundleParams) WithTimeout(timeout time.Duration) *DownloadSupportBundleParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the download support bundle params
func (o *DownloadSupportBundleParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the download support bundle params
func (o *DownloadSupportBundleParams) WithContext(ctx context.Context) *DownloadSupportBundleParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the download support bundle params
func (o *DownloadSupportBundleParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the download support bundle params
func (o *DownloadSupportBundleParams) WithHTTPClient(client *http.Client) *DownloadSupportBundleParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the download support bundle params
func (o *DownloadSupportBundleParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCUUID adds the cUUID to the download support bundle params
func (o *DownloadSupportBundleParams) WithCUUID(cUUID strfmt.UUID) *DownloadSupportBundleParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the download support bundle params
func (o *DownloadSupportBundleParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WithSbUUID adds the sbUUID to the download support bundle params
func (o *DownloadSupportBundleParams) WithSbUUID(sbUUID strfmt.UUID) *DownloadSupportBundleParams {
	o.SetSbUUID(sbUUID)
	return o
}

// SetSbUUID adds the sbUuid to the download support bundle params
func (o *DownloadSupportBundleParams) SetSbUUID(sbUUID strfmt.UUID) {
	o.SbUUID = sbUUID
}

// WithUniUUID adds the uniUUID to the download support bundle params
func (o *DownloadSupportBundleParams) WithUniUUID(uniUUID strfmt.UUID) *DownloadSupportBundleParams {
	o.SetUniUUID(uniUUID)
	return o
}

// SetUniUUID adds the uniUuid to the download support bundle params
func (o *DownloadSupportBundleParams) SetUniUUID(uniUUID strfmt.UUID) {
	o.UniUUID = uniUUID
}

// WriteToRequest writes these params to a swagger request
func (o *DownloadSupportBundleParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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