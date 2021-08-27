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

	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
)

// NewCreateAlertDestinationParams creates a new CreateAlertDestinationParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewCreateAlertDestinationParams() *CreateAlertDestinationParams {
	return &CreateAlertDestinationParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewCreateAlertDestinationParamsWithTimeout creates a new CreateAlertDestinationParams object
// with the ability to set a timeout on a request.
func NewCreateAlertDestinationParamsWithTimeout(timeout time.Duration) *CreateAlertDestinationParams {
	return &CreateAlertDestinationParams{
		timeout: timeout,
	}
}

// NewCreateAlertDestinationParamsWithContext creates a new CreateAlertDestinationParams object
// with the ability to set a context for a request.
func NewCreateAlertDestinationParamsWithContext(ctx context.Context) *CreateAlertDestinationParams {
	return &CreateAlertDestinationParams{
		Context: ctx,
	}
}

// NewCreateAlertDestinationParamsWithHTTPClient creates a new CreateAlertDestinationParams object
// with the ability to set a custom HTTPClient for a request.
func NewCreateAlertDestinationParamsWithHTTPClient(client *http.Client) *CreateAlertDestinationParams {
	return &CreateAlertDestinationParams{
		HTTPClient: client,
	}
}

/* CreateAlertDestinationParams contains all the parameters to send to the API endpoint
   for the create alert destination operation.

   Typically these are written to a http.Request.
*/
type CreateAlertDestinationParams struct {

	// CreateAlertDestinationRequest.
	CreateAlertDestinationRequest *models.AlertDestinationFormData

	// CUUID.
	//
	// Format: uuid
	CUUID strfmt.UUID

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the create alert destination params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *CreateAlertDestinationParams) WithDefaults() *CreateAlertDestinationParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the create alert destination params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *CreateAlertDestinationParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the create alert destination params
func (o *CreateAlertDestinationParams) WithTimeout(timeout time.Duration) *CreateAlertDestinationParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the create alert destination params
func (o *CreateAlertDestinationParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the create alert destination params
func (o *CreateAlertDestinationParams) WithContext(ctx context.Context) *CreateAlertDestinationParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the create alert destination params
func (o *CreateAlertDestinationParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the create alert destination params
func (o *CreateAlertDestinationParams) WithHTTPClient(client *http.Client) *CreateAlertDestinationParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the create alert destination params
func (o *CreateAlertDestinationParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCreateAlertDestinationRequest adds the createAlertDestinationRequest to the create alert destination params
func (o *CreateAlertDestinationParams) WithCreateAlertDestinationRequest(createAlertDestinationRequest *models.AlertDestinationFormData) *CreateAlertDestinationParams {
	o.SetCreateAlertDestinationRequest(createAlertDestinationRequest)
	return o
}

// SetCreateAlertDestinationRequest adds the createAlertDestinationRequest to the create alert destination params
func (o *CreateAlertDestinationParams) SetCreateAlertDestinationRequest(createAlertDestinationRequest *models.AlertDestinationFormData) {
	o.CreateAlertDestinationRequest = createAlertDestinationRequest
}

// WithCUUID adds the cUUID to the create alert destination params
func (o *CreateAlertDestinationParams) WithCUUID(cUUID strfmt.UUID) *CreateAlertDestinationParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the create alert destination params
func (o *CreateAlertDestinationParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WriteToRequest writes these params to a swagger request
func (o *CreateAlertDestinationParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error
	if o.CreateAlertDestinationRequest != nil {
		if err := r.SetBodyParam(o.CreateAlertDestinationRequest); err != nil {
			return err
		}
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