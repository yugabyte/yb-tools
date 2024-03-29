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

// NewDeleteKMSConfigParams creates a new DeleteKMSConfigParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewDeleteKMSConfigParams() *DeleteKMSConfigParams {
	return &DeleteKMSConfigParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewDeleteKMSConfigParamsWithTimeout creates a new DeleteKMSConfigParams object
// with the ability to set a timeout on a request.
func NewDeleteKMSConfigParamsWithTimeout(timeout time.Duration) *DeleteKMSConfigParams {
	return &DeleteKMSConfigParams{
		timeout: timeout,
	}
}

// NewDeleteKMSConfigParamsWithContext creates a new DeleteKMSConfigParams object
// with the ability to set a context for a request.
func NewDeleteKMSConfigParamsWithContext(ctx context.Context) *DeleteKMSConfigParams {
	return &DeleteKMSConfigParams{
		Context: ctx,
	}
}

// NewDeleteKMSConfigParamsWithHTTPClient creates a new DeleteKMSConfigParams object
// with the ability to set a custom HTTPClient for a request.
func NewDeleteKMSConfigParamsWithHTTPClient(client *http.Client) *DeleteKMSConfigParams {
	return &DeleteKMSConfigParams{
		HTTPClient: client,
	}
}

/* DeleteKMSConfigParams contains all the parameters to send to the API endpoint
   for the delete k m s config operation.

   Typically these are written to a http.Request.
*/
type DeleteKMSConfigParams struct {

	// CUUID.
	//
	// Format: uuid
	CUUID strfmt.UUID

	// ConfigUUID.
	//
	// Format: uuid
	ConfigUUID strfmt.UUID

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the delete k m s config params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteKMSConfigParams) WithDefaults() *DeleteKMSConfigParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the delete k m s config params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteKMSConfigParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the delete k m s config params
func (o *DeleteKMSConfigParams) WithTimeout(timeout time.Duration) *DeleteKMSConfigParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the delete k m s config params
func (o *DeleteKMSConfigParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the delete k m s config params
func (o *DeleteKMSConfigParams) WithContext(ctx context.Context) *DeleteKMSConfigParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the delete k m s config params
func (o *DeleteKMSConfigParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the delete k m s config params
func (o *DeleteKMSConfigParams) WithHTTPClient(client *http.Client) *DeleteKMSConfigParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the delete k m s config params
func (o *DeleteKMSConfigParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCUUID adds the cUUID to the delete k m s config params
func (o *DeleteKMSConfigParams) WithCUUID(cUUID strfmt.UUID) *DeleteKMSConfigParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the delete k m s config params
func (o *DeleteKMSConfigParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WithConfigUUID adds the configUUID to the delete k m s config params
func (o *DeleteKMSConfigParams) WithConfigUUID(configUUID strfmt.UUID) *DeleteKMSConfigParams {
	o.SetConfigUUID(configUUID)
	return o
}

// SetConfigUUID adds the configUuid to the delete k m s config params
func (o *DeleteKMSConfigParams) SetConfigUUID(configUUID strfmt.UUID) {
	o.ConfigUUID = configUUID
}

// WriteToRequest writes these params to a swagger request
func (o *DeleteKMSConfigParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param cUUID
	if err := r.SetPathParam("cUUID", o.CUUID.String()); err != nil {
		return err
	}

	// path param configUUID
	if err := r.SetPathParam("configUUID", o.ConfigUUID.String()); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
