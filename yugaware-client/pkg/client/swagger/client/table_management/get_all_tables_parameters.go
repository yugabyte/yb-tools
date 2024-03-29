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
	"github.com/go-openapi/swag"
)

// NewGetAllTablesParams creates a new GetAllTablesParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewGetAllTablesParams() *GetAllTablesParams {
	return &GetAllTablesParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewGetAllTablesParamsWithTimeout creates a new GetAllTablesParams object
// with the ability to set a timeout on a request.
func NewGetAllTablesParamsWithTimeout(timeout time.Duration) *GetAllTablesParams {
	return &GetAllTablesParams{
		timeout: timeout,
	}
}

// NewGetAllTablesParamsWithContext creates a new GetAllTablesParams object
// with the ability to set a context for a request.
func NewGetAllTablesParamsWithContext(ctx context.Context) *GetAllTablesParams {
	return &GetAllTablesParams{
		Context: ctx,
	}
}

// NewGetAllTablesParamsWithHTTPClient creates a new GetAllTablesParams object
// with the ability to set a custom HTTPClient for a request.
func NewGetAllTablesParamsWithHTTPClient(client *http.Client) *GetAllTablesParams {
	return &GetAllTablesParams{
		HTTPClient: client,
	}
}

/* GetAllTablesParams contains all the parameters to send to the API endpoint
   for the get all tables operation.

   Typically these are written to a http.Request.
*/
type GetAllTablesParams struct {

	// CUUID.
	//
	// Format: uuid
	CUUID strfmt.UUID

	// IncludeParentTableInfo.
	IncludeParentTableInfo *bool

	// UniUUID.
	//
	// Format: uuid
	UniUUID strfmt.UUID

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the get all tables params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetAllTablesParams) WithDefaults() *GetAllTablesParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the get all tables params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetAllTablesParams) SetDefaults() {
	var (
		includeParentTableInfoDefault = bool(false)
	)

	val := GetAllTablesParams{
		IncludeParentTableInfo: &includeParentTableInfoDefault,
	}

	val.timeout = o.timeout
	val.Context = o.Context
	val.HTTPClient = o.HTTPClient
	*o = val
}

// WithTimeout adds the timeout to the get all tables params
func (o *GetAllTablesParams) WithTimeout(timeout time.Duration) *GetAllTablesParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get all tables params
func (o *GetAllTablesParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get all tables params
func (o *GetAllTablesParams) WithContext(ctx context.Context) *GetAllTablesParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get all tables params
func (o *GetAllTablesParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get all tables params
func (o *GetAllTablesParams) WithHTTPClient(client *http.Client) *GetAllTablesParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get all tables params
func (o *GetAllTablesParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCUUID adds the cUUID to the get all tables params
func (o *GetAllTablesParams) WithCUUID(cUUID strfmt.UUID) *GetAllTablesParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the get all tables params
func (o *GetAllTablesParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WithIncludeParentTableInfo adds the includeParentTableInfo to the get all tables params
func (o *GetAllTablesParams) WithIncludeParentTableInfo(includeParentTableInfo *bool) *GetAllTablesParams {
	o.SetIncludeParentTableInfo(includeParentTableInfo)
	return o
}

// SetIncludeParentTableInfo adds the includeParentTableInfo to the get all tables params
func (o *GetAllTablesParams) SetIncludeParentTableInfo(includeParentTableInfo *bool) {
	o.IncludeParentTableInfo = includeParentTableInfo
}

// WithUniUUID adds the uniUUID to the get all tables params
func (o *GetAllTablesParams) WithUniUUID(uniUUID strfmt.UUID) *GetAllTablesParams {
	o.SetUniUUID(uniUUID)
	return o
}

// SetUniUUID adds the uniUuid to the get all tables params
func (o *GetAllTablesParams) SetUniUUID(uniUUID strfmt.UUID) {
	o.UniUUID = uniUUID
}

// WriteToRequest writes these params to a swagger request
func (o *GetAllTablesParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param cUUID
	if err := r.SetPathParam("cUUID", o.CUUID.String()); err != nil {
		return err
	}

	if o.IncludeParentTableInfo != nil {

		// query param includeParentTableInfo
		var qrIncludeParentTableInfo bool

		if o.IncludeParentTableInfo != nil {
			qrIncludeParentTableInfo = *o.IncludeParentTableInfo
		}
		qIncludeParentTableInfo := swag.FormatBool(qrIncludeParentTableInfo)
		if qIncludeParentTableInfo != "" {

			if err := r.SetQueryParam("includeParentTableInfo", qIncludeParentTableInfo); err != nil {
				return err
			}
		}
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
