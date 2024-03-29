// Code generated by go-swagger; DO NOT EDIT.

package instance_types

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

	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
)

// NewCreateInstanceTypeParams creates a new CreateInstanceTypeParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewCreateInstanceTypeParams() *CreateInstanceTypeParams {
	return &CreateInstanceTypeParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewCreateInstanceTypeParamsWithTimeout creates a new CreateInstanceTypeParams object
// with the ability to set a timeout on a request.
func NewCreateInstanceTypeParamsWithTimeout(timeout time.Duration) *CreateInstanceTypeParams {
	return &CreateInstanceTypeParams{
		timeout: timeout,
	}
}

// NewCreateInstanceTypeParamsWithContext creates a new CreateInstanceTypeParams object
// with the ability to set a context for a request.
func NewCreateInstanceTypeParamsWithContext(ctx context.Context) *CreateInstanceTypeParams {
	return &CreateInstanceTypeParams{
		Context: ctx,
	}
}

// NewCreateInstanceTypeParamsWithHTTPClient creates a new CreateInstanceTypeParams object
// with the ability to set a custom HTTPClient for a request.
func NewCreateInstanceTypeParamsWithHTTPClient(client *http.Client) *CreateInstanceTypeParams {
	return &CreateInstanceTypeParams{
		HTTPClient: client,
	}
}

/* CreateInstanceTypeParams contains all the parameters to send to the API endpoint
   for the create instance type operation.

   Typically these are written to a http.Request.
*/
type CreateInstanceTypeParams struct {

	/* InstanceType.

	   Instance type data of the instance to be stored
	*/
	InstanceType *models.InstanceType

	// CUUID.
	//
	// Format: uuid
	CUUID strfmt.UUID

	// PUUID.
	//
	// Format: uuid
	PUUID strfmt.UUID

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the create instance type params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *CreateInstanceTypeParams) WithDefaults() *CreateInstanceTypeParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the create instance type params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *CreateInstanceTypeParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the create instance type params
func (o *CreateInstanceTypeParams) WithTimeout(timeout time.Duration) *CreateInstanceTypeParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the create instance type params
func (o *CreateInstanceTypeParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the create instance type params
func (o *CreateInstanceTypeParams) WithContext(ctx context.Context) *CreateInstanceTypeParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the create instance type params
func (o *CreateInstanceTypeParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the create instance type params
func (o *CreateInstanceTypeParams) WithHTTPClient(client *http.Client) *CreateInstanceTypeParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the create instance type params
func (o *CreateInstanceTypeParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithInstanceType adds the instanceType to the create instance type params
func (o *CreateInstanceTypeParams) WithInstanceType(instanceType *models.InstanceType) *CreateInstanceTypeParams {
	o.SetInstanceType(instanceType)
	return o
}

// SetInstanceType adds the instanceType to the create instance type params
func (o *CreateInstanceTypeParams) SetInstanceType(instanceType *models.InstanceType) {
	o.InstanceType = instanceType
}

// WithCUUID adds the cUUID to the create instance type params
func (o *CreateInstanceTypeParams) WithCUUID(cUUID strfmt.UUID) *CreateInstanceTypeParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the create instance type params
func (o *CreateInstanceTypeParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WithPUUID adds the pUUID to the create instance type params
func (o *CreateInstanceTypeParams) WithPUUID(pUUID strfmt.UUID) *CreateInstanceTypeParams {
	o.SetPUUID(pUUID)
	return o
}

// SetPUUID adds the pUuid to the create instance type params
func (o *CreateInstanceTypeParams) SetPUUID(pUUID strfmt.UUID) {
	o.PUUID = pUUID
}

// WriteToRequest writes these params to a swagger request
func (o *CreateInstanceTypeParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error
	if o.InstanceType != nil {
		if err := r.SetBodyParam(o.InstanceType); err != nil {
			return err
		}
	}

	// path param cUUID
	if err := r.SetPathParam("cUUID", o.CUUID.String()); err != nil {
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
