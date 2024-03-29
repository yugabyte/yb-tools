// Code generated by go-swagger; DO NOT EDIT.

package availability_zones

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

// NewCreateAZParams creates a new CreateAZParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewCreateAZParams() *CreateAZParams {
	return &CreateAZParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewCreateAZParamsWithTimeout creates a new CreateAZParams object
// with the ability to set a timeout on a request.
func NewCreateAZParamsWithTimeout(timeout time.Duration) *CreateAZParams {
	return &CreateAZParams{
		timeout: timeout,
	}
}

// NewCreateAZParamsWithContext creates a new CreateAZParams object
// with the ability to set a context for a request.
func NewCreateAZParamsWithContext(ctx context.Context) *CreateAZParams {
	return &CreateAZParams{
		Context: ctx,
	}
}

// NewCreateAZParamsWithHTTPClient creates a new CreateAZParams object
// with the ability to set a custom HTTPClient for a request.
func NewCreateAZParamsWithHTTPClient(client *http.Client) *CreateAZParams {
	return &CreateAZParams{
		HTTPClient: client,
	}
}

/* CreateAZParams contains all the parameters to send to the API endpoint
   for the create a z operation.

   Typically these are written to a http.Request.
*/
type CreateAZParams struct {

	/* AzFormData.

	   Availability zone form data
	*/
	AzFormData *models.AvailabilityZoneFormData

	// CUUID.
	//
	// Format: uuid
	CUUID strfmt.UUID

	// PUUID.
	//
	// Format: uuid
	PUUID strfmt.UUID

	// RUUID.
	//
	// Format: uuid
	RUUID strfmt.UUID

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the create a z params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *CreateAZParams) WithDefaults() *CreateAZParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the create a z params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *CreateAZParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the create a z params
func (o *CreateAZParams) WithTimeout(timeout time.Duration) *CreateAZParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the create a z params
func (o *CreateAZParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the create a z params
func (o *CreateAZParams) WithContext(ctx context.Context) *CreateAZParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the create a z params
func (o *CreateAZParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the create a z params
func (o *CreateAZParams) WithHTTPClient(client *http.Client) *CreateAZParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the create a z params
func (o *CreateAZParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithAzFormData adds the azFormData to the create a z params
func (o *CreateAZParams) WithAzFormData(azFormData *models.AvailabilityZoneFormData) *CreateAZParams {
	o.SetAzFormData(azFormData)
	return o
}

// SetAzFormData adds the azFormData to the create a z params
func (o *CreateAZParams) SetAzFormData(azFormData *models.AvailabilityZoneFormData) {
	o.AzFormData = azFormData
}

// WithCUUID adds the cUUID to the create a z params
func (o *CreateAZParams) WithCUUID(cUUID strfmt.UUID) *CreateAZParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the create a z params
func (o *CreateAZParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WithPUUID adds the pUUID to the create a z params
func (o *CreateAZParams) WithPUUID(pUUID strfmt.UUID) *CreateAZParams {
	o.SetPUUID(pUUID)
	return o
}

// SetPUUID adds the pUuid to the create a z params
func (o *CreateAZParams) SetPUUID(pUUID strfmt.UUID) {
	o.PUUID = pUUID
}

// WithRUUID adds the rUUID to the create a z params
func (o *CreateAZParams) WithRUUID(rUUID strfmt.UUID) *CreateAZParams {
	o.SetRUUID(rUUID)
	return o
}

// SetRUUID adds the rUuid to the create a z params
func (o *CreateAZParams) SetRUUID(rUUID strfmt.UUID) {
	o.RUUID = rUUID
}

// WriteToRequest writes these params to a swagger request
func (o *CreateAZParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error
	if o.AzFormData != nil {
		if err := r.SetBodyParam(o.AzFormData); err != nil {
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

	// path param rUUID
	if err := r.SetPathParam("rUUID", o.RUUID.String()); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
