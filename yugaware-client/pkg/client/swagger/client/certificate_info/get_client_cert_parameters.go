// Code generated by go-swagger; DO NOT EDIT.

package certificate_info

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

// NewGetClientCertParams creates a new GetClientCertParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewGetClientCertParams() *GetClientCertParams {
	return &GetClientCertParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewGetClientCertParamsWithTimeout creates a new GetClientCertParams object
// with the ability to set a timeout on a request.
func NewGetClientCertParamsWithTimeout(timeout time.Duration) *GetClientCertParams {
	return &GetClientCertParams{
		timeout: timeout,
	}
}

// NewGetClientCertParamsWithContext creates a new GetClientCertParams object
// with the ability to set a context for a request.
func NewGetClientCertParamsWithContext(ctx context.Context) *GetClientCertParams {
	return &GetClientCertParams{
		Context: ctx,
	}
}

// NewGetClientCertParamsWithHTTPClient creates a new GetClientCertParams object
// with the ability to set a custom HTTPClient for a request.
func NewGetClientCertParamsWithHTTPClient(client *http.Client) *GetClientCertParams {
	return &GetClientCertParams{
		HTTPClient: client,
	}
}

/* GetClientCertParams contains all the parameters to send to the API endpoint
   for the get client cert operation.

   Typically these are written to a http.Request.
*/
type GetClientCertParams struct {

	// CUUID.
	//
	// Format: uuid
	CUUID strfmt.UUID

	/* Certificate.

	   post certificate info
	*/
	Certificate *models.ClientCertParams

	// RUUID.
	//
	// Format: uuid
	RUUID strfmt.UUID

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the get client cert params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetClientCertParams) WithDefaults() *GetClientCertParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the get client cert params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetClientCertParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the get client cert params
func (o *GetClientCertParams) WithTimeout(timeout time.Duration) *GetClientCertParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get client cert params
func (o *GetClientCertParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get client cert params
func (o *GetClientCertParams) WithContext(ctx context.Context) *GetClientCertParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get client cert params
func (o *GetClientCertParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get client cert params
func (o *GetClientCertParams) WithHTTPClient(client *http.Client) *GetClientCertParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get client cert params
func (o *GetClientCertParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCUUID adds the cUUID to the get client cert params
func (o *GetClientCertParams) WithCUUID(cUUID strfmt.UUID) *GetClientCertParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the get client cert params
func (o *GetClientCertParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WithCertificate adds the certificate to the get client cert params
func (o *GetClientCertParams) WithCertificate(certificate *models.ClientCertParams) *GetClientCertParams {
	o.SetCertificate(certificate)
	return o
}

// SetCertificate adds the certificate to the get client cert params
func (o *GetClientCertParams) SetCertificate(certificate *models.ClientCertParams) {
	o.Certificate = certificate
}

// WithRUUID adds the rUUID to the get client cert params
func (o *GetClientCertParams) WithRUUID(rUUID strfmt.UUID) *GetClientCertParams {
	o.SetRUUID(rUUID)
	return o
}

// SetRUUID adds the rUuid to the get client cert params
func (o *GetClientCertParams) SetRUUID(rUUID strfmt.UUID) {
	o.RUUID = rUUID
}

// WriteToRequest writes these params to a swagger request
func (o *GetClientCertParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param cUUID
	if err := r.SetPathParam("cUUID", o.CUUID.String()); err != nil {
		return err
	}
	if o.Certificate != nil {
		if err := r.SetBodyParam(o.Certificate); err != nil {
			return err
		}
	}

	// path param rUUID
	if err := r.SetPathParam("rUUID", o.RUUID.String()); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
