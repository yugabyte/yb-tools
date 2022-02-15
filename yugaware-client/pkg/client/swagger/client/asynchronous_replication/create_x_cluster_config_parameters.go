// Code generated by go-swagger; DO NOT EDIT.

package asynchronous_replication

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

// NewCreateXClusterConfigParams creates a new CreateXClusterConfigParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewCreateXClusterConfigParams() *CreateXClusterConfigParams {
	return &CreateXClusterConfigParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewCreateXClusterConfigParamsWithTimeout creates a new CreateXClusterConfigParams object
// with the ability to set a timeout on a request.
func NewCreateXClusterConfigParamsWithTimeout(timeout time.Duration) *CreateXClusterConfigParams {
	return &CreateXClusterConfigParams{
		timeout: timeout,
	}
}

// NewCreateXClusterConfigParamsWithContext creates a new CreateXClusterConfigParams object
// with the ability to set a context for a request.
func NewCreateXClusterConfigParamsWithContext(ctx context.Context) *CreateXClusterConfigParams {
	return &CreateXClusterConfigParams{
		Context: ctx,
	}
}

// NewCreateXClusterConfigParamsWithHTTPClient creates a new CreateXClusterConfigParams object
// with the ability to set a custom HTTPClient for a request.
func NewCreateXClusterConfigParamsWithHTTPClient(client *http.Client) *CreateXClusterConfigParams {
	return &CreateXClusterConfigParams{
		HTTPClient: client,
	}
}

/* CreateXClusterConfigParams contains all the parameters to send to the API endpoint
   for the create x cluster config operation.

   Typically these are written to a http.Request.
*/
type CreateXClusterConfigParams struct {

	// CUUID.
	//
	// Format: uuid
	CUUID strfmt.UUID

	/* XclusterReplicationCreateFormData.

	   XCluster Replication Create Form Data
	*/
	XclusterReplicationCreateFormData *models.XClusterConfigCreateFormData

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the create x cluster config params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *CreateXClusterConfigParams) WithDefaults() *CreateXClusterConfigParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the create x cluster config params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *CreateXClusterConfigParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the create x cluster config params
func (o *CreateXClusterConfigParams) WithTimeout(timeout time.Duration) *CreateXClusterConfigParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the create x cluster config params
func (o *CreateXClusterConfigParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the create x cluster config params
func (o *CreateXClusterConfigParams) WithContext(ctx context.Context) *CreateXClusterConfigParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the create x cluster config params
func (o *CreateXClusterConfigParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the create x cluster config params
func (o *CreateXClusterConfigParams) WithHTTPClient(client *http.Client) *CreateXClusterConfigParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the create x cluster config params
func (o *CreateXClusterConfigParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCUUID adds the cUUID to the create x cluster config params
func (o *CreateXClusterConfigParams) WithCUUID(cUUID strfmt.UUID) *CreateXClusterConfigParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the create x cluster config params
func (o *CreateXClusterConfigParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WithXclusterReplicationCreateFormData adds the xclusterReplicationCreateFormData to the create x cluster config params
func (o *CreateXClusterConfigParams) WithXclusterReplicationCreateFormData(xclusterReplicationCreateFormData *models.XClusterConfigCreateFormData) *CreateXClusterConfigParams {
	o.SetXclusterReplicationCreateFormData(xclusterReplicationCreateFormData)
	return o
}

// SetXclusterReplicationCreateFormData adds the xclusterReplicationCreateFormData to the create x cluster config params
func (o *CreateXClusterConfigParams) SetXclusterReplicationCreateFormData(xclusterReplicationCreateFormData *models.XClusterConfigCreateFormData) {
	o.XclusterReplicationCreateFormData = xclusterReplicationCreateFormData
}

// WriteToRequest writes these params to a swagger request
func (o *CreateXClusterConfigParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param cUUID
	if err := r.SetPathParam("cUUID", o.CUUID.String()); err != nil {
		return err
	}
	if o.XclusterReplicationCreateFormData != nil {
		if err := r.SetBodyParam(o.XclusterReplicationCreateFormData); err != nil {
			return err
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
