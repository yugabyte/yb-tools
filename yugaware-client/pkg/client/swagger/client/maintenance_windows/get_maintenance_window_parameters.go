// Code generated by go-swagger; DO NOT EDIT.

package maintenance_windows

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

// NewGetMaintenanceWindowParams creates a new GetMaintenanceWindowParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewGetMaintenanceWindowParams() *GetMaintenanceWindowParams {
	return &GetMaintenanceWindowParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewGetMaintenanceWindowParamsWithTimeout creates a new GetMaintenanceWindowParams object
// with the ability to set a timeout on a request.
func NewGetMaintenanceWindowParamsWithTimeout(timeout time.Duration) *GetMaintenanceWindowParams {
	return &GetMaintenanceWindowParams{
		timeout: timeout,
	}
}

// NewGetMaintenanceWindowParamsWithContext creates a new GetMaintenanceWindowParams object
// with the ability to set a context for a request.
func NewGetMaintenanceWindowParamsWithContext(ctx context.Context) *GetMaintenanceWindowParams {
	return &GetMaintenanceWindowParams{
		Context: ctx,
	}
}

// NewGetMaintenanceWindowParamsWithHTTPClient creates a new GetMaintenanceWindowParams object
// with the ability to set a custom HTTPClient for a request.
func NewGetMaintenanceWindowParamsWithHTTPClient(client *http.Client) *GetMaintenanceWindowParams {
	return &GetMaintenanceWindowParams{
		HTTPClient: client,
	}
}

/* GetMaintenanceWindowParams contains all the parameters to send to the API endpoint
   for the get maintenance window operation.

   Typically these are written to a http.Request.
*/
type GetMaintenanceWindowParams struct {

	// CUUID.
	//
	// Format: uuid
	CUUID strfmt.UUID

	// WindowUUID.
	//
	// Format: uuid
	WindowUUID strfmt.UUID

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the get maintenance window params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetMaintenanceWindowParams) WithDefaults() *GetMaintenanceWindowParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the get maintenance window params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetMaintenanceWindowParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the get maintenance window params
func (o *GetMaintenanceWindowParams) WithTimeout(timeout time.Duration) *GetMaintenanceWindowParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get maintenance window params
func (o *GetMaintenanceWindowParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get maintenance window params
func (o *GetMaintenanceWindowParams) WithContext(ctx context.Context) *GetMaintenanceWindowParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get maintenance window params
func (o *GetMaintenanceWindowParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get maintenance window params
func (o *GetMaintenanceWindowParams) WithHTTPClient(client *http.Client) *GetMaintenanceWindowParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get maintenance window params
func (o *GetMaintenanceWindowParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCUUID adds the cUUID to the get maintenance window params
func (o *GetMaintenanceWindowParams) WithCUUID(cUUID strfmt.UUID) *GetMaintenanceWindowParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the get maintenance window params
func (o *GetMaintenanceWindowParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WithWindowUUID adds the windowUUID to the get maintenance window params
func (o *GetMaintenanceWindowParams) WithWindowUUID(windowUUID strfmt.UUID) *GetMaintenanceWindowParams {
	o.SetWindowUUID(windowUUID)
	return o
}

// SetWindowUUID adds the windowUuid to the get maintenance window params
func (o *GetMaintenanceWindowParams) SetWindowUUID(windowUUID strfmt.UUID) {
	o.WindowUUID = windowUUID
}

// WriteToRequest writes these params to a swagger request
func (o *GetMaintenanceWindowParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param cUUID
	if err := r.SetPathParam("cUUID", o.CUUID.String()); err != nil {
		return err
	}

	// path param windowUUID
	if err := r.SetPathParam("windowUUID", o.WindowUUID.String()); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
