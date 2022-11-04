// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"encoding/json"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// BackupStorageInfo Backup Storage Info for doing restore operation
//
// swagger:model BackupStorageInfo
type BackupStorageInfo struct {

	// Backup type
	// Enum: [YQL_TABLE_TYPE REDIS_TABLE_TYPE PGSQL_TABLE_TYPE TRANSACTION_STATUS_TABLE_TYPE]
	BackupType string `json:"backupType,omitempty"`

	// Keyspace name
	Keyspace string `json:"keyspace,omitempty"`

	// Is SSE
	Sse bool `json:"sse,omitempty"`

	// Storage location
	StorageLocation string `json:"storageLocation,omitempty"`

	// Tables
	TableNameList []string `json:"tableNameList"`
}

// Validate validates this backup storage info
func (m *BackupStorageInfo) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateBackupType(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

var backupStorageInfoTypeBackupTypePropEnum []interface{}

func init() {
	var res []string
	if err := json.Unmarshal([]byte(`["YQL_TABLE_TYPE","REDIS_TABLE_TYPE","PGSQL_TABLE_TYPE","TRANSACTION_STATUS_TABLE_TYPE"]`), &res); err != nil {
		panic(err)
	}
	for _, v := range res {
		backupStorageInfoTypeBackupTypePropEnum = append(backupStorageInfoTypeBackupTypePropEnum, v)
	}
}

const (

	// BackupStorageInfoBackupTypeYQLTABLETYPE captures enum value "YQL_TABLE_TYPE"
	BackupStorageInfoBackupTypeYQLTABLETYPE string = "YQL_TABLE_TYPE"

	// BackupStorageInfoBackupTypeREDISTABLETYPE captures enum value "REDIS_TABLE_TYPE"
	BackupStorageInfoBackupTypeREDISTABLETYPE string = "REDIS_TABLE_TYPE"

	// BackupStorageInfoBackupTypePGSQLTABLETYPE captures enum value "PGSQL_TABLE_TYPE"
	BackupStorageInfoBackupTypePGSQLTABLETYPE string = "PGSQL_TABLE_TYPE"

	// BackupStorageInfoBackupTypeTRANSACTIONSTATUSTABLETYPE captures enum value "TRANSACTION_STATUS_TABLE_TYPE"
	BackupStorageInfoBackupTypeTRANSACTIONSTATUSTABLETYPE string = "TRANSACTION_STATUS_TABLE_TYPE"
)

// prop value enum
func (m *BackupStorageInfo) validateBackupTypeEnum(path, location string, value string) error {
	if err := validate.EnumCase(path, location, value, backupStorageInfoTypeBackupTypePropEnum, true); err != nil {
		return err
	}
	return nil
}

func (m *BackupStorageInfo) validateBackupType(formats strfmt.Registry) error {
	if swag.IsZero(m.BackupType) { // not required
		return nil
	}

	// value enum
	if err := m.validateBackupTypeEnum("backupType", "body", m.BackupType); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this backup storage info based on context it is used
func (m *BackupStorageInfo) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *BackupStorageInfo) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *BackupStorageInfo) UnmarshalBinary(b []byte) error {
	var res BackupStorageInfo
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
