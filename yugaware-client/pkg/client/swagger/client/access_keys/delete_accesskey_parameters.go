// Code generated by go-swagger; DO NOT EDIT.

package access_keys

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

// NewDeleteAccesskeyParams creates a new DeleteAccesskeyParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewDeleteAccesskeyParams() *DeleteAccesskeyParams {
	return &DeleteAccesskeyParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewDeleteAccesskeyParamsWithTimeout creates a new DeleteAccesskeyParams object
// with the ability to set a timeout on a request.
func NewDeleteAccesskeyParamsWithTimeout(timeout time.Duration) *DeleteAccesskeyParams {
	return &DeleteAccesskeyParams{
		timeout: timeout,
	}
}

// NewDeleteAccesskeyParamsWithContext creates a new DeleteAccesskeyParams object
// with the ability to set a context for a request.
func NewDeleteAccesskeyParamsWithContext(ctx context.Context) *DeleteAccesskeyParams {
	return &DeleteAccesskeyParams{
		Context: ctx,
	}
}

// NewDeleteAccesskeyParamsWithHTTPClient creates a new DeleteAccesskeyParams object
// with the ability to set a custom HTTPClient for a request.
func NewDeleteAccesskeyParamsWithHTTPClient(client *http.Client) *DeleteAccesskeyParams {
	return &DeleteAccesskeyParams{
		HTTPClient: client,
	}
}

/* DeleteAccesskeyParams contains all the parameters to send to the API endpoint
   for the delete accesskey operation.

   Typically these are written to a http.Request.
*/
type DeleteAccesskeyParams struct {

	// CUUID.
	//
	// Format: uuid
	CUUID strfmt.UUID

	// KeyCode.
	KeyCode string

	// PUUID.
	//
	// Format: uuid
	PUUID strfmt.UUID

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the delete accesskey params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteAccesskeyParams) WithDefaults() *DeleteAccesskeyParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the delete accesskey params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteAccesskeyParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the delete accesskey params
func (o *DeleteAccesskeyParams) WithTimeout(timeout time.Duration) *DeleteAccesskeyParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the delete accesskey params
func (o *DeleteAccesskeyParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the delete accesskey params
func (o *DeleteAccesskeyParams) WithContext(ctx context.Context) *DeleteAccesskeyParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the delete accesskey params
func (o *DeleteAccesskeyParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the delete accesskey params
func (o *DeleteAccesskeyParams) WithHTTPClient(client *http.Client) *DeleteAccesskeyParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the delete accesskey params
func (o *DeleteAccesskeyParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCUUID adds the cUUID to the delete accesskey params
func (o *DeleteAccesskeyParams) WithCUUID(cUUID strfmt.UUID) *DeleteAccesskeyParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the delete accesskey params
func (o *DeleteAccesskeyParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WithKeyCode adds the keyCode to the delete accesskey params
func (o *DeleteAccesskeyParams) WithKeyCode(keyCode string) *DeleteAccesskeyParams {
	o.SetKeyCode(keyCode)
	return o
}

// SetKeyCode adds the keyCode to the delete accesskey params
func (o *DeleteAccesskeyParams) SetKeyCode(keyCode string) {
	o.KeyCode = keyCode
}

// WithPUUID adds the pUUID to the delete accesskey params
func (o *DeleteAccesskeyParams) WithPUUID(pUUID strfmt.UUID) *DeleteAccesskeyParams {
	o.SetPUUID(pUUID)
	return o
}

// SetPUUID adds the pUuid to the delete accesskey params
func (o *DeleteAccesskeyParams) SetPUUID(pUUID strfmt.UUID) {
	o.PUUID = pUUID
}

// WriteToRequest writes these params to a swagger request
func (o *DeleteAccesskeyParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param cUUID
	if err := r.SetPathParam("cUUID", o.CUUID.String()); err != nil {
		return err
	}

	// path param keyCode
	if err := r.SetPathParam("keyCode", o.KeyCode); err != nil {
		return err
	}

	// path param pUUID
	if err := r.SetPathParam("pUUID", o.PUUID.String()); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
