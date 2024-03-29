// Code generated by go-swagger; DO NOT EDIT.

package universe_upgrades_management

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

// NewRestartUniverseParams creates a new RestartUniverseParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewRestartUniverseParams() *RestartUniverseParams {
	return &RestartUniverseParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewRestartUniverseParamsWithTimeout creates a new RestartUniverseParams object
// with the ability to set a timeout on a request.
func NewRestartUniverseParamsWithTimeout(timeout time.Duration) *RestartUniverseParams {
	return &RestartUniverseParams{
		timeout: timeout,
	}
}

// NewRestartUniverseParamsWithContext creates a new RestartUniverseParams object
// with the ability to set a context for a request.
func NewRestartUniverseParamsWithContext(ctx context.Context) *RestartUniverseParams {
	return &RestartUniverseParams{
		Context: ctx,
	}
}

// NewRestartUniverseParamsWithHTTPClient creates a new RestartUniverseParams object
// with the ability to set a custom HTTPClient for a request.
func NewRestartUniverseParamsWithHTTPClient(client *http.Client) *RestartUniverseParams {
	return &RestartUniverseParams{
		HTTPClient: client,
	}
}

/* RestartUniverseParams contains all the parameters to send to the API endpoint
   for the restart universe operation.

   Typically these are written to a http.Request.
*/
type RestartUniverseParams struct {

	// CUUID.
	//
	// Format: uuid
	CUUID strfmt.UUID

	// UniUUID.
	//
	// Format: uuid
	UniUUID strfmt.UUID

	/* UpgradeTaskParams.

	   Upgrade Task Params
	*/
	UpgradeTaskParams *models.UpgradeTaskParams

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the restart universe params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *RestartUniverseParams) WithDefaults() *RestartUniverseParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the restart universe params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *RestartUniverseParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the restart universe params
func (o *RestartUniverseParams) WithTimeout(timeout time.Duration) *RestartUniverseParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the restart universe params
func (o *RestartUniverseParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the restart universe params
func (o *RestartUniverseParams) WithContext(ctx context.Context) *RestartUniverseParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the restart universe params
func (o *RestartUniverseParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the restart universe params
func (o *RestartUniverseParams) WithHTTPClient(client *http.Client) *RestartUniverseParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the restart universe params
func (o *RestartUniverseParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCUUID adds the cUUID to the restart universe params
func (o *RestartUniverseParams) WithCUUID(cUUID strfmt.UUID) *RestartUniverseParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the restart universe params
func (o *RestartUniverseParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WithUniUUID adds the uniUUID to the restart universe params
func (o *RestartUniverseParams) WithUniUUID(uniUUID strfmt.UUID) *RestartUniverseParams {
	o.SetUniUUID(uniUUID)
	return o
}

// SetUniUUID adds the uniUuid to the restart universe params
func (o *RestartUniverseParams) SetUniUUID(uniUUID strfmt.UUID) {
	o.UniUUID = uniUUID
}

// WithUpgradeTaskParams adds the upgradeTaskParams to the restart universe params
func (o *RestartUniverseParams) WithUpgradeTaskParams(upgradeTaskParams *models.UpgradeTaskParams) *RestartUniverseParams {
	o.SetUpgradeTaskParams(upgradeTaskParams)
	return o
}

// SetUpgradeTaskParams adds the upgradeTaskParams to the restart universe params
func (o *RestartUniverseParams) SetUpgradeTaskParams(upgradeTaskParams *models.UpgradeTaskParams) {
	o.UpgradeTaskParams = upgradeTaskParams
}

// WriteToRequest writes these params to a swagger request
func (o *RestartUniverseParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
	if o.UpgradeTaskParams != nil {
		if err := r.SetBodyParam(o.UpgradeTaskParams); err != nil {
			return err
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
