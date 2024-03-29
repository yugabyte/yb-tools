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

// NewCreatebackupScheduleParams creates a new CreatebackupScheduleParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewCreatebackupScheduleParams() *CreatebackupScheduleParams {
	return &CreatebackupScheduleParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewCreatebackupScheduleParamsWithTimeout creates a new CreatebackupScheduleParams object
// with the ability to set a timeout on a request.
func NewCreatebackupScheduleParamsWithTimeout(timeout time.Duration) *CreatebackupScheduleParams {
	return &CreatebackupScheduleParams{
		timeout: timeout,
	}
}

// NewCreatebackupScheduleParamsWithContext creates a new CreatebackupScheduleParams object
// with the ability to set a context for a request.
func NewCreatebackupScheduleParamsWithContext(ctx context.Context) *CreatebackupScheduleParams {
	return &CreatebackupScheduleParams{
		Context: ctx,
	}
}

// NewCreatebackupScheduleParamsWithHTTPClient creates a new CreatebackupScheduleParams object
// with the ability to set a custom HTTPClient for a request.
func NewCreatebackupScheduleParamsWithHTTPClient(client *http.Client) *CreatebackupScheduleParams {
	return &CreatebackupScheduleParams{
		HTTPClient: client,
	}
}

/* CreatebackupScheduleParams contains all the parameters to send to the API endpoint
   for the createbackup schedule operation.

   Typically these are written to a http.Request.
*/
type CreatebackupScheduleParams struct {

	/* Backup.

	   Parameters of the backup to be restored
	*/
	Backup *models.BackupRequestParams

	// CUUID.
	//
	// Format: uuid
	CUUID strfmt.UUID

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the createbackup schedule params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *CreatebackupScheduleParams) WithDefaults() *CreatebackupScheduleParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the createbackup schedule params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *CreatebackupScheduleParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the createbackup schedule params
func (o *CreatebackupScheduleParams) WithTimeout(timeout time.Duration) *CreatebackupScheduleParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the createbackup schedule params
func (o *CreatebackupScheduleParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the createbackup schedule params
func (o *CreatebackupScheduleParams) WithContext(ctx context.Context) *CreatebackupScheduleParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the createbackup schedule params
func (o *CreatebackupScheduleParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the createbackup schedule params
func (o *CreatebackupScheduleParams) WithHTTPClient(client *http.Client) *CreatebackupScheduleParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the createbackup schedule params
func (o *CreatebackupScheduleParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithBackup adds the backup to the createbackup schedule params
func (o *CreatebackupScheduleParams) WithBackup(backup *models.BackupRequestParams) *CreatebackupScheduleParams {
	o.SetBackup(backup)
	return o
}

// SetBackup adds the backup to the createbackup schedule params
func (o *CreatebackupScheduleParams) SetBackup(backup *models.BackupRequestParams) {
	o.Backup = backup
}

// WithCUUID adds the cUUID to the createbackup schedule params
func (o *CreatebackupScheduleParams) WithCUUID(cUUID strfmt.UUID) *CreatebackupScheduleParams {
	o.SetCUUID(cUUID)
	return o
}

// SetCUUID adds the cUuid to the createbackup schedule params
func (o *CreatebackupScheduleParams) SetCUUID(cUUID strfmt.UUID) {
	o.CUUID = cUUID
}

// WriteToRequest writes these params to a swagger request
func (o *CreatebackupScheduleParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
