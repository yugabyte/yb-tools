// Code generated by go-swagger; DO NOT EDIT.

package alerts

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

// NewSendTestAlertParams creates a new SendTestAlertParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewSendTestAlertParams() *SendTestAlertParams {
	return &SendTestAlertParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewSendTestAlertParamsWithTimeout creates a new SendTestAlertParams object
// with the ability to set a timeout on a request.
func NewSendTestAlertParamsWithTimeout(timeout time.Duration) *SendTestAlertParams {
	return &SendTestAlertParams{
		timeout: timeout,
	}
}

// NewSendTestAlertParamsWithContext creates a new SendTestAlertParams object
// with the ability to set a context for a request.
func NewSendTestAlertParamsWithContext(ctx context.Context) *SendTestAlertParams {
	return &SendTestAlertParams{
		Context: ctx,
	}
}

// NewSendTestAlertParamsWithHTTPClient creates a new SendTestAlertParams object
// with the ability to set a custom HTTPClient for a request.
func NewSendTestAlertParamsWithHTTPClient(client *http.Client) *SendTestAlertParams {
	return &SendTestAlertParams{
		HTTPClient: client,
	}
}

/* SendTestAlertParams contains all the parameters to send to the API endpoint
   for the send test alert operation.

   Typically these are written to a http.Request.
*/
type SendTestAlertParams struct {

	// CUUID.
	//
	// Format: uuid
	CUUID strfmt.UUID

	// ConfigurationUUID.
	//
	// Format: uuid
	ConfigurationUUID strfmt.UUID

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the send test alert params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *SendTestAlertParams) WithDefaults() *SendTestAlertParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the send test alert params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *SendTestAlertParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the send test alert params
func (o *SendTestAlertParams) WithTimeout(timeout time.Duration) *SendTestAlertParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the send test alert params
func (o *SendTestAlertParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the send test alert params
func (o *SendTestAlertParams) WithContext(ctx context.Context) *SendTestAlertParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the send test alert params
func (o *SendTestAlertParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the send test alert params
func (o *SendTestAlertParams) WithHTTPClient(client *http.Client) *SendTestAlertParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the send test alert params
func (o *SendTestAlertParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCUUID adds the cUUID to the send test alert params
func (o *SendTestAlertParams) WithCUUID(cUUID strfmt.UUID) *SendTestAlertParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the send test alert params
func (o *SendTestAlertParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WithConfigurationUUID adds the configurationUUID to the send test alert params
func (o *SendTestAlertParams) WithConfigurationUUID(configurationUUID strfmt.UUID) *SendTestAlertParams {
	o.SetConfigurationUUID(configurationUUID)
	return o
}

// SetConfigurationUUID adds the configurationUuid to the send test alert params
func (o *SendTestAlertParams) SetConfigurationUUID(configurationUUID strfmt.UUID) {
	o.ConfigurationUUID = configurationUUID
}

// WriteToRequest writes these params to a swagger request
func (o *SendTestAlertParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param cUUID
	if err := r.SetPathParam("cUUID", o.CUUID.String()); err != nil {
		return err
	}

	// path param configurationUUID
	if err := r.SetPathParam("configurationUUID", o.ConfigurationUUID.String()); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
