// Code generated by go-swagger; DO NOT EDIT.

package table_management

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

// NewDescribeTableParams creates a new DescribeTableParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewDescribeTableParams() *DescribeTableParams {
	return &DescribeTableParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewDescribeTableParamsWithTimeout creates a new DescribeTableParams object
// with the ability to set a timeout on a request.
func NewDescribeTableParamsWithTimeout(timeout time.Duration) *DescribeTableParams {
	return &DescribeTableParams{
		timeout: timeout,
	}
}

// NewDescribeTableParamsWithContext creates a new DescribeTableParams object
// with the ability to set a context for a request.
func NewDescribeTableParamsWithContext(ctx context.Context) *DescribeTableParams {
	return &DescribeTableParams{
		Context: ctx,
	}
}

// NewDescribeTableParamsWithHTTPClient creates a new DescribeTableParams object
// with the ability to set a custom HTTPClient for a request.
func NewDescribeTableParamsWithHTTPClient(client *http.Client) *DescribeTableParams {
	return &DescribeTableParams{
		HTTPClient: client,
	}
}

/* DescribeTableParams contains all the parameters to send to the API endpoint
   for the describe table operation.

   Typically these are written to a http.Request.
*/
type DescribeTableParams struct {

	// CUUID.
	//
	// Format: uuid
	CUUID strfmt.UUID

	// TableUUID.
	//
	// Format: uuid
	TableUUID strfmt.UUID

	// UniUUID.
	//
	// Format: uuid
	UniUUID strfmt.UUID

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the describe table params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DescribeTableParams) WithDefaults() *DescribeTableParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the describe table params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DescribeTableParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the describe table params
func (o *DescribeTableParams) WithTimeout(timeout time.Duration) *DescribeTableParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the describe table params
func (o *DescribeTableParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the describe table params
func (o *DescribeTableParams) WithContext(ctx context.Context) *DescribeTableParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the describe table params
func (o *DescribeTableParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the describe table params
func (o *DescribeTableParams) WithHTTPClient(client *http.Client) *DescribeTableParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the describe table params
func (o *DescribeTableParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCUUID adds the cUUID to the describe table params
func (o *DescribeTableParams) WithCUUID(cUUID strfmt.UUID) *DescribeTableParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the describe table params
func (o *DescribeTableParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WithTableUUID adds the tableUUID to the describe table params
func (o *DescribeTableParams) WithTableUUID(tableUUID strfmt.UUID) *DescribeTableParams {
	o.SetTableUUID(tableUUID)
	return o
}

// SetTableUUID adds the tableUuid to the describe table params
func (o *DescribeTableParams) SetTableUUID(tableUUID strfmt.UUID) {
	o.TableUUID = tableUUID
}

// WithUniUUID adds the uniUUID to the describe table params
func (o *DescribeTableParams) WithUniUUID(uniUUID strfmt.UUID) *DescribeTableParams {
	o.SetUniUUID(uniUUID)
	return o
}

// SetUniUUID adds the uniUuid to the describe table params
func (o *DescribeTableParams) SetUniUUID(uniUUID strfmt.UUID) {
	o.UniUUID = uniUUID
}

// WriteToRequest writes these params to a swagger request
func (o *DescribeTableParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param cUUID
	if err := r.SetPathParam("cUUID", o.CUUID.String()); err != nil {
		return err
	}

	// path param tableUUID
	if err := r.SetPathParam("tableUUID", o.TableUUID.String()); err != nil {
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