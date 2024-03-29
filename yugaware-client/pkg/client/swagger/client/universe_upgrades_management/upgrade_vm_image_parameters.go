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

// NewUpgradeVMImageParams creates a new UpgradeVMImageParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewUpgradeVMImageParams() *UpgradeVMImageParams {
	return &UpgradeVMImageParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewUpgradeVMImageParamsWithTimeout creates a new UpgradeVMImageParams object
// with the ability to set a timeout on a request.
func NewUpgradeVMImageParamsWithTimeout(timeout time.Duration) *UpgradeVMImageParams {
	return &UpgradeVMImageParams{
		timeout: timeout,
	}
}

// NewUpgradeVMImageParamsWithContext creates a new UpgradeVMImageParams object
// with the ability to set a context for a request.
func NewUpgradeVMImageParamsWithContext(ctx context.Context) *UpgradeVMImageParams {
	return &UpgradeVMImageParams{
		Context: ctx,
	}
}

// NewUpgradeVMImageParamsWithHTTPClient creates a new UpgradeVMImageParams object
// with the ability to set a custom HTTPClient for a request.
func NewUpgradeVMImageParamsWithHTTPClient(client *http.Client) *UpgradeVMImageParams {
	return &UpgradeVMImageParams{
		HTTPClient: client,
	}
}

/* UpgradeVMImageParams contains all the parameters to send to the API endpoint
   for the upgrade VM image operation.

   Typically these are written to a http.Request.
*/
type UpgradeVMImageParams struct {

	// CUUID.
	//
	// Format: uuid
	CUUID strfmt.UUID

	// UniUUID.
	//
	// Format: uuid
	UniUUID strfmt.UUID

	/* VmimageUpgradeParams.

	   VM Image Upgrade Params
	*/
	VmimageUpgradeParams *models.VMImageUpgradeParams

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the upgrade VM image params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *UpgradeVMImageParams) WithDefaults() *UpgradeVMImageParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the upgrade VM image params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *UpgradeVMImageParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the upgrade VM image params
func (o *UpgradeVMImageParams) WithTimeout(timeout time.Duration) *UpgradeVMImageParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the upgrade VM image params
func (o *UpgradeVMImageParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the upgrade VM image params
func (o *UpgradeVMImageParams) WithContext(ctx context.Context) *UpgradeVMImageParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the upgrade VM image params
func (o *UpgradeVMImageParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the upgrade VM image params
func (o *UpgradeVMImageParams) WithHTTPClient(client *http.Client) *UpgradeVMImageParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the upgrade VM image params
func (o *UpgradeVMImageParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCUUID adds the cUUID to the upgrade VM image params
func (o *UpgradeVMImageParams) WithCUUID(cUUID strfmt.UUID) *UpgradeVMImageParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the upgrade VM image params
func (o *UpgradeVMImageParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WithUniUUID adds the uniUUID to the upgrade VM image params
func (o *UpgradeVMImageParams) WithUniUUID(uniUUID strfmt.UUID) *UpgradeVMImageParams {
	o.SetUniUUID(uniUUID)
	return o
}

// SetUniUUID adds the uniUuid to the upgrade VM image params
func (o *UpgradeVMImageParams) SetUniUUID(uniUUID strfmt.UUID) {
	o.UniUUID = uniUUID
}

// WithVmimageUpgradeParams adds the vmimageUpgradeParams to the upgrade VM image params
func (o *UpgradeVMImageParams) WithVmimageUpgradeParams(vmimageUpgradeParams *models.VMImageUpgradeParams) *UpgradeVMImageParams {
	o.SetVmimageUpgradeParams(vmimageUpgradeParams)
	return o
}

// SetVmimageUpgradeParams adds the vmimageUpgradeParams to the upgrade VM image params
func (o *UpgradeVMImageParams) SetVmimageUpgradeParams(vmimageUpgradeParams *models.VMImageUpgradeParams) {
	o.VmimageUpgradeParams = vmimageUpgradeParams
}

// WriteToRequest writes these params to a swagger request
func (o *UpgradeVMImageParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
	if o.VmimageUpgradeParams != nil {
		if err := r.SetBodyParam(o.VmimageUpgradeParams); err != nil {
			return err
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
