// Code generated by go-swagger; DO NOT EDIT.

package backups

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

// NewRestoreParams creates a new RestoreParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewRestoreParams() *RestoreParams {
	return &RestoreParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewRestoreParamsWithTimeout creates a new RestoreParams object
// with the ability to set a timeout on a request.
func NewRestoreParamsWithTimeout(timeout time.Duration) *RestoreParams {
	return &RestoreParams{
		timeout: timeout,
	}
}

// NewRestoreParamsWithContext creates a new RestoreParams object
// with the ability to set a context for a request.
func NewRestoreParamsWithContext(ctx context.Context) *RestoreParams {
	return &RestoreParams{
		Context: ctx,
	}
}

// NewRestoreParamsWithHTTPClient creates a new RestoreParams object
// with the ability to set a custom HTTPClient for a request.
func NewRestoreParamsWithHTTPClient(client *http.Client) *RestoreParams {
	return &RestoreParams{
		HTTPClient: client,
	}
}

/* RestoreParams contains all the parameters to send to the API endpoint
   for the restore operation.

   Typically these are written to a http.Request.
*/
type RestoreParams struct {

	/* Backup.

	   Parameters of the backup to be restored
	*/
	Backup *models.BackupTableParams

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

// WithDefaults hydrates default values in the restore params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *RestoreParams) WithDefaults() *RestoreParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the restore params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *RestoreParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the restore params
func (o *RestoreParams) WithTimeout(timeout time.Duration) *RestoreParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the restore params
func (o *RestoreParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the restore params
func (o *RestoreParams) WithContext(ctx context.Context) *RestoreParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the restore params
func (o *RestoreParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the restore params
func (o *RestoreParams) WithHTTPClient(client *http.Client) *RestoreParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the restore params
func (o *RestoreParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithBackup adds the backup to the restore params
func (o *RestoreParams) WithBackup(backup *models.BackupTableParams) *RestoreParams {
	o.SetBackup(backup)
	return o
}

// SetBackup adds the backup to the restore params
func (o *RestoreParams) SetBackup(backup *models.BackupTableParams) {
	o.Backup = backup
}

// WithCUUID adds the cUUID to the restore params
func (o *RestoreParams) WithCUUID(cUUID strfmt.UUID) *RestoreParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the restore params
func (o *RestoreParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WithUniUUID adds the uniUUID to the restore params
func (o *RestoreParams) WithUniUUID(uniUUID strfmt.UUID) *RestoreParams {
	o.SetUniUUID(uniUUID)
	return o
}

// SetUniUUID adds the uniUuid to the restore params
func (o *RestoreParams) SetUniUUID(uniUUID strfmt.UUID) {
	o.UniUUID = uniUUID
}

// WriteToRequest writes these params to a swagger request
func (o *RestoreParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error
	if o.Backup != nil {
		if err := r.SetBodyParam(o.Backup); err != nil {
			return err
		}
	}

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