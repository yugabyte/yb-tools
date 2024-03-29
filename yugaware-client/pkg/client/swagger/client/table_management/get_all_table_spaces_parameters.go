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

// NewGetAllTableSpacesParams creates a new GetAllTableSpacesParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewGetAllTableSpacesParams() *GetAllTableSpacesParams {
	return &GetAllTableSpacesParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewGetAllTableSpacesParamsWithTimeout creates a new GetAllTableSpacesParams object
// with the ability to set a timeout on a request.
func NewGetAllTableSpacesParamsWithTimeout(timeout time.Duration) *GetAllTableSpacesParams {
	return &GetAllTableSpacesParams{
		timeout: timeout,
	}
}

// NewGetAllTableSpacesParamsWithContext creates a new GetAllTableSpacesParams object
// with the ability to set a context for a request.
func NewGetAllTableSpacesParamsWithContext(ctx context.Context) *GetAllTableSpacesParams {
	return &GetAllTableSpacesParams{
		Context: ctx,
	}
}

// NewGetAllTableSpacesParamsWithHTTPClient creates a new GetAllTableSpacesParams object
// with the ability to set a custom HTTPClient for a request.
func NewGetAllTableSpacesParamsWithHTTPClient(client *http.Client) *GetAllTableSpacesParams {
	return &GetAllTableSpacesParams{
		HTTPClient: client,
	}
}

/* GetAllTableSpacesParams contains all the parameters to send to the API endpoint
   for the get all table spaces operation.

   Typically these are written to a http.Request.
*/
type GetAllTableSpacesParams struct {

	// CUUID.
	//
	// Format: uuid
	CUUID strfmt.UUID

	// UniUUID.
	//
	// Format: uuid
	UniUUID strfmt.UUID

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the get all table spaces params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetAllTableSpacesParams) WithDefaults() *GetAllTableSpacesParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the get all table spaces params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetAllTableSpacesParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the get all table spaces params
func (o *GetAllTableSpacesParams) WithTimeout(timeout time.Duration) *GetAllTableSpacesParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get all table spaces params
func (o *GetAllTableSpacesParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get all table spaces params
func (o *GetAllTableSpacesParams) WithContext(ctx context.Context) *GetAllTableSpacesParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get all table spaces params
func (o *GetAllTableSpacesParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get all table spaces params
func (o *GetAllTableSpacesParams) WithHTTPClient(client *http.Client) *GetAllTableSpacesParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get all table spaces params
func (o *GetAllTableSpacesParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCUUID adds the cUUID to the get all table spaces params
func (o *GetAllTableSpacesParams) WithCUUID(cUUID strfmt.UUID) *GetAllTableSpacesParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the get all table spaces params
func (o *GetAllTableSpacesParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WithUniUUID adds the uniUUID to the get all table spaces params
func (o *GetAllTableSpacesParams) WithUniUUID(uniUUID strfmt.UUID) *GetAllTableSpacesParams {
	o.SetUniUUID(uniUUID)
	return o
}

// SetUniUUID adds the uniUuid to the get all table spaces params
func (o *GetAllTableSpacesParams) SetUniUUID(uniUUID strfmt.UUID) {
	o.UniUUID = uniUUID
}

// WriteToRequest writes these params to a swagger request
func (o *GetAllTableSpacesParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param cUUID
	if err := r.SetPathParam("cUUID", o.CUUID.String()); err != nil {
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
