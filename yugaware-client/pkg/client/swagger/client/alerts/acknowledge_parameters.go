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

// NewAcknowledgeParams creates a new AcknowledgeParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewAcknowledgeParams() *AcknowledgeParams {
	return &AcknowledgeParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewAcknowledgeParamsWithTimeout creates a new AcknowledgeParams object
// with the ability to set a timeout on a request.
func NewAcknowledgeParamsWithTimeout(timeout time.Duration) *AcknowledgeParams {
	return &AcknowledgeParams{
		timeout: timeout,
	}
}

// NewAcknowledgeParamsWithContext creates a new AcknowledgeParams object
// with the ability to set a context for a request.
func NewAcknowledgeParamsWithContext(ctx context.Context) *AcknowledgeParams {
	return &AcknowledgeParams{
		Context: ctx,
	}
}

// NewAcknowledgeParamsWithHTTPClient creates a new AcknowledgeParams object
// with the ability to set a custom HTTPClient for a request.
func NewAcknowledgeParamsWithHTTPClient(client *http.Client) *AcknowledgeParams {
	return &AcknowledgeParams{
		HTTPClient: client,
	}
}

/* AcknowledgeParams contains all the parameters to send to the API endpoint
   for the acknowledge operation.

   Typically these are written to a http.Request.
*/
type AcknowledgeParams struct {

	// AlertUUID.
	//
	// Format: uuid
	AlertUUID strfmt.UUID

	// CUUID.
	//
	// Format: uuid
	CUUID strfmt.UUID

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the acknowledge params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *AcknowledgeParams) WithDefaults() *AcknowledgeParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the acknowledge params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *AcknowledgeParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the acknowledge params
func (o *AcknowledgeParams) WithTimeout(timeout time.Duration) *AcknowledgeParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the acknowledge params
func (o *AcknowledgeParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the acknowledge params
func (o *AcknowledgeParams) WithContext(ctx context.Context) *AcknowledgeParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the acknowledge params
func (o *AcknowledgeParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the acknowledge params
func (o *AcknowledgeParams) WithHTTPClient(client *http.Client) *AcknowledgeParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the acknowledge params
func (o *AcknowledgeParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithAlertUUID adds the alertUUID to the acknowledge params
func (o *AcknowledgeParams) WithAlertUUID(alertUUID strfmt.UUID) *AcknowledgeParams {
	o.SetAlertUUID(alertUUID)
	return o
}

// SetAlertUUID adds the alertUuid to the acknowledge params
func (o *AcknowledgeParams) SetAlertUUID(alertUUID strfmt.UUID) {
	o.AlertUUID = alertUUID
}

// WithCUUID adds the cUUID to the acknowledge params
func (o *AcknowledgeParams) WithCUUID(cUUID strfmt.UUID) *AcknowledgeParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the acknowledge params
func (o *AcknowledgeParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WriteToRequest writes these params to a swagger request
func (o *AcknowledgeParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param alertUUID
	if err := r.SetPathParam("alertUUID", o.AlertUUID.String()); err != nil {
		return err
	}

	// path param cUUID
	if err := r.SetPathParam("cUUID", o.CUUID.String()); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}