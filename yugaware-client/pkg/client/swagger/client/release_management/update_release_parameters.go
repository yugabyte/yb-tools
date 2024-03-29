// Code generated by go-swagger; DO NOT EDIT.

package release_management

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

// NewUpdateReleaseParams creates a new UpdateReleaseParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewUpdateReleaseParams() *UpdateReleaseParams {
	return &UpdateReleaseParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewUpdateReleaseParamsWithTimeout creates a new UpdateReleaseParams object
// with the ability to set a timeout on a request.
func NewUpdateReleaseParamsWithTimeout(timeout time.Duration) *UpdateReleaseParams {
	return &UpdateReleaseParams{
		timeout: timeout,
	}
}

// NewUpdateReleaseParamsWithContext creates a new UpdateReleaseParams object
// with the ability to set a context for a request.
func NewUpdateReleaseParamsWithContext(ctx context.Context) *UpdateReleaseParams {
	return &UpdateReleaseParams{
		Context: ctx,
	}
}

// NewUpdateReleaseParamsWithHTTPClient creates a new UpdateReleaseParams object
// with the ability to set a custom HTTPClient for a request.
func NewUpdateReleaseParamsWithHTTPClient(client *http.Client) *UpdateReleaseParams {
	return &UpdateReleaseParams{
		HTTPClient: client,
	}
}

/* UpdateReleaseParams contains all the parameters to send to the API endpoint
   for the update release operation.

   Typically these are written to a http.Request.
*/
type UpdateReleaseParams struct {

	/* Release.

	   Release data to be updated
	*/
	Release interface{}

	// CUUID.
	//
	// Format: uuid
	CUUID strfmt.UUID

	// Name.
	Name string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the update release params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *UpdateReleaseParams) WithDefaults() *UpdateReleaseParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the update release params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *UpdateReleaseParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the update release params
func (o *UpdateReleaseParams) WithTimeout(timeout time.Duration) *UpdateReleaseParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the update release params
func (o *UpdateReleaseParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the update release params
func (o *UpdateReleaseParams) WithContext(ctx context.Context) *UpdateReleaseParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the update release params
func (o *UpdateReleaseParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the update release params
func (o *UpdateReleaseParams) WithHTTPClient(client *http.Client) *UpdateReleaseParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the update release params
func (o *UpdateReleaseParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithRelease adds the release to the update release params
func (o *UpdateReleaseParams) WithRelease(release interface{}) *UpdateReleaseParams {
	o.SetRelease(release)
	return o
}

// SetRelease adds the release to the update release params
func (o *UpdateReleaseParams) SetRelease(release interface{}) {
	o.Release = release
}

// WithCUUID adds the cUUID to the update release params
func (o *UpdateReleaseParams) WithCUUID(cUUID strfmt.UUID) *UpdateReleaseParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the update release params
func (o *UpdateReleaseParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WithName adds the name to the update release params
func (o *UpdateReleaseParams) WithName(name string) *UpdateReleaseParams {
	o.SetName(name)
	return o
}

// SetName adds the name to the update release params
func (o *UpdateReleaseParams) SetName(name string) {
	o.Name = name
}

// WriteToRequest writes these params to a swagger request
func (o *UpdateReleaseParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error
	if o.Release != nil {
		if err := r.SetBodyParam(o.Release); err != nil {
			return err
		}
	}

	// path param cUUID
	if err := r.SetPathParam("cUUID", o.CUUID.String()); err != nil {
		return err
	}

	// path param name
	if err := r.SetPathParam("name", o.Name); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
