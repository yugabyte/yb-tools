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

// NewListAlertDestinationsParams creates a new ListAlertDestinationsParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListAlertDestinationsParams() *ListAlertDestinationsParams {
	return &ListAlertDestinationsParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListAlertDestinationsParamsWithTimeout creates a new ListAlertDestinationsParams object
// with the ability to set a timeout on a request.
func NewListAlertDestinationsParamsWithTimeout(timeout time.Duration) *ListAlertDestinationsParams {
	return &ListAlertDestinationsParams{
		timeout: timeout,
	}
}

// NewListAlertDestinationsParamsWithContext creates a new ListAlertDestinationsParams object
// with the ability to set a context for a request.
func NewListAlertDestinationsParamsWithContext(ctx context.Context) *ListAlertDestinationsParams {
	return &ListAlertDestinationsParams{
		Context: ctx,
	}
}

// NewListAlertDestinationsParamsWithHTTPClient creates a new ListAlertDestinationsParams object
// with the ability to set a custom HTTPClient for a request.
func NewListAlertDestinationsParamsWithHTTPClient(client *http.Client) *ListAlertDestinationsParams {
	return &ListAlertDestinationsParams{
		HTTPClient: client,
	}
}

/* ListAlertDestinationsParams contains all the parameters to send to the API endpoint
   for the list alert destinations operation.

   Typically these are written to a http.Request.
*/
type ListAlertDestinationsParams struct {

	// CUUID.
	//
	// Format: uuid
	CUUID strfmt.UUID

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list alert destinations params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListAlertDestinationsParams) WithDefaults() *ListAlertDestinationsParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list alert destinations params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListAlertDestinationsParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list alert destinations params
func (o *ListAlertDestinationsParams) WithTimeout(timeout time.Duration) *ListAlertDestinationsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list alert destinations params
func (o *ListAlertDestinationsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list alert destinations params
func (o *ListAlertDestinationsParams) WithContext(ctx context.Context) *ListAlertDestinationsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list alert destinations params
func (o *ListAlertDestinationsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list alert destinations params
func (o *ListAlertDestinationsParams) WithHTTPClient(client *http.Client) *ListAlertDestinationsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list alert destinations params
func (o *ListAlertDestinationsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCUUID adds the cUUID to the list alert destinations params
func (o *ListAlertDestinationsParams) WithCUUID(cUUID strfmt.UUID) *ListAlertDestinationsParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the list alert destinations params
func (o *ListAlertDestinationsParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WriteToRequest writes these params to a swagger request
func (o *ListAlertDestinationsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param cUUID
	if err := r.SetPathParam("cUUID", o.CUUID.String()); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
