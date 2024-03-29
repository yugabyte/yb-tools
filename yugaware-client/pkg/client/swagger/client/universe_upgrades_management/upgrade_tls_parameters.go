// Code generated by go-swagger; DO NOT EDIT.

package universe_upgrades_management

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

	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
)

// NewUpgradeTLSParams creates a new UpgradeTLSParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewUpgradeTLSParams() *UpgradeTLSParams {
	return &UpgradeTLSParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewUpgradeTLSParamsWithTimeout creates a new UpgradeTLSParams object
// with the ability to set a timeout on a request.
func NewUpgradeTLSParamsWithTimeout(timeout time.Duration) *UpgradeTLSParams {
	return &UpgradeTLSParams{
		timeout: timeout,
	}
}

// NewUpgradeTLSParamsWithContext creates a new UpgradeTLSParams object
// with the ability to set a context for a request.
func NewUpgradeTLSParamsWithContext(ctx context.Context) *UpgradeTLSParams {
	return &UpgradeTLSParams{
		Context: ctx,
	}
}

// NewUpgradeTLSParamsWithHTTPClient creates a new UpgradeTLSParams object
// with the ability to set a custom HTTPClient for a request.
func NewUpgradeTLSParamsWithHTTPClient(client *http.Client) *UpgradeTLSParams {
	return &UpgradeTLSParams{
		HTTPClient: client,
	}
}

/* UpgradeTLSParams contains all the parameters to send to the API endpoint
   for the upgrade Tls operation.

   Typically these are written to a http.Request.
*/
type UpgradeTLSParams struct {

	// CUUID.
	//
	// Format: uuid
	CUUID strfmt.UUID

	/* TLSToggleParams.

	   TLS Toggle Params
	*/
	TLSToggleParams *models.TLSToggleParams

	// UniUUID.
	//
	// Format: uuid
	UniUUID strfmt.UUID

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the upgrade Tls params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *UpgradeTLSParams) WithDefaults() *UpgradeTLSParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the upgrade Tls params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *UpgradeTLSParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the upgrade Tls params
func (o *UpgradeTLSParams) WithTimeout(timeout time.Duration) *UpgradeTLSParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the upgrade Tls params
func (o *UpgradeTLSParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the upgrade Tls params
func (o *UpgradeTLSParams) WithContext(ctx context.Context) *UpgradeTLSParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the upgrade Tls params
func (o *UpgradeTLSParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the upgrade Tls params
func (o *UpgradeTLSParams) WithHTTPClient(client *http.Client) *UpgradeTLSParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the upgrade Tls params
func (o *UpgradeTLSParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCUUID adds the cUUID to the upgrade Tls params
func (o *UpgradeTLSParams) WithCUUID(cUUID strfmt.UUID) *UpgradeTLSParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the upgrade Tls params
func (o *UpgradeTLSParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WithTLSToggleParams adds the tLSToggleParams to the upgrade Tls params
func (o *UpgradeTLSParams) WithTLSToggleParams(tLSToggleParams *models.TLSToggleParams) *UpgradeTLSParams {
	o.SetTLSToggleParams(tLSToggleParams)
	return o
}

// SetTLSToggleParams adds the tlsToggleParams to the upgrade Tls params
func (o *UpgradeTLSParams) SetTLSToggleParams(tLSToggleParams *models.TLSToggleParams) {
	o.TLSToggleParams = tLSToggleParams
}

// WithUniUUID adds the uniUUID to the upgrade Tls params
func (o *UpgradeTLSParams) WithUniUUID(uniUUID strfmt.UUID) *UpgradeTLSParams {
	o.SetUniUUID(uniUUID)
	return o
}

// SetUniUUID adds the uniUuid to the upgrade Tls params
func (o *UpgradeTLSParams) SetUniUUID(uniUUID strfmt.UUID) {
	o.UniUUID = uniUUID
}

// WriteToRequest writes these params to a swagger request
func (o *UpgradeTLSParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param cUUID
	if err := r.SetPathParam("cUUID", o.CUUID.String()); err != nil {
		return err
	}
	if o.TLSToggleParams != nil {
		if err := r.SetBodyParam(o.TLSToggleParams); err != nil {
			return err
		}
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
