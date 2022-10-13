// Code generated by go-swagger; DO NOT EDIT.

package customer_configuration

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

// NewEditCustomerConfigParams creates a new EditCustomerConfigParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewEditCustomerConfigParams() *EditCustomerConfigParams {
	return &EditCustomerConfigParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewEditCustomerConfigParamsWithTimeout creates a new EditCustomerConfigParams object
// with the ability to set a timeout on a request.
func NewEditCustomerConfigParamsWithTimeout(timeout time.Duration) *EditCustomerConfigParams {
	return &EditCustomerConfigParams{
		timeout: timeout,
	}
}

// NewEditCustomerConfigParamsWithContext creates a new EditCustomerConfigParams object
// with the ability to set a context for a request.
func NewEditCustomerConfigParamsWithContext(ctx context.Context) *EditCustomerConfigParams {
	return &EditCustomerConfigParams{
		Context: ctx,
	}
}

// NewEditCustomerConfigParamsWithHTTPClient creates a new EditCustomerConfigParams object
// with the ability to set a custom HTTPClient for a request.
func NewEditCustomerConfigParamsWithHTTPClient(client *http.Client) *EditCustomerConfigParams {
	return &EditCustomerConfigParams{
		HTTPClient: client,
	}
}

/* EditCustomerConfigParams contains all the parameters to send to the API endpoint
   for the edit customer config operation.

   Typically these are written to a http.Request.
*/
type EditCustomerConfigParams struct {

	/* Config.

	   Configuration data to be updated
	*/
	Config *models.CustomerConfig

	// CUUID.
	//
	// Format: uuid
	CUUID strfmt.UUID

	// ConfigUUID.
	//
	// Format: uuid
	ConfigUUID strfmt.UUID

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the edit customer config params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *EditCustomerConfigParams) WithDefaults() *EditCustomerConfigParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the edit customer config params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *EditCustomerConfigParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the edit customer config params
func (o *EditCustomerConfigParams) WithTimeout(timeout time.Duration) *EditCustomerConfigParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the edit customer config params
func (o *EditCustomerConfigParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the edit customer config params
func (o *EditCustomerConfigParams) WithContext(ctx context.Context) *EditCustomerConfigParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the edit customer config params
func (o *EditCustomerConfigParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the edit customer config params
func (o *EditCustomerConfigParams) WithHTTPClient(client *http.Client) *EditCustomerConfigParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the edit customer config params
func (o *EditCustomerConfigParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithConfig adds the config to the edit customer config params
func (o *EditCustomerConfigParams) WithConfig(config *models.CustomerConfig) *EditCustomerConfigParams {
	o.SetConfig(config)
	return o
}

// SetConfig adds the config to the edit customer config params
func (o *EditCustomerConfigParams) SetConfig(config *models.CustomerConfig) {
	o.Config = config
}

// WithCUUID adds the cUUID to the edit customer config params
func (o *EditCustomerConfigParams) WithCUUID(cUUID strfmt.UUID) *EditCustomerConfigParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the edit customer config params
func (o *EditCustomerConfigParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WithConfigUUID adds the configUUID to the edit customer config params
func (o *EditCustomerConfigParams) WithConfigUUID(configUUID strfmt.UUID) *EditCustomerConfigParams {
	o.SetConfigUUID(configUUID)
	return o
}

// SetConfigUUID adds the configUuid to the edit customer config params
func (o *EditCustomerConfigParams) SetConfigUUID(configUUID strfmt.UUID) {
	o.ConfigUUID = configUUID
}

// WriteToRequest writes these params to a swagger request
func (o *EditCustomerConfigParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error
	if o.Config != nil {
		if err := r.SetBodyParam(o.Config); err != nil {
			return err
		}
	}

	// path param cUUID
	if err := r.SetPathParam("cUUID", o.CUUID.String()); err != nil {
		return err
	}

	// path param configUUID
	if err := r.SetPathParam("configUUID", o.ConfigUUID.String()); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}