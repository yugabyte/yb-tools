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

// EditBackupParams Edit backup parameters
//
// swagger:model EditBackupParams
type EditBackupParams struct {

	// Time unit for backup expiry
	// Enum: [NANOSECONDS MICROSECONDS MILLISECONDS SECONDS MINUTES HOURS DAYS MONTHS YEARS]
	ExpiryTimeUnit string `json:"expiryTimeUnit,omitempty"`

	// New backup Storage config
	// Format: uuid
	StorageConfigUUID strfmt.UUID `json:"storageConfigUUID,omitempty"`

	// Time before deleting the backup from storage, in milliseconds
	TimeBeforeDeleteFromPresentInMillis int64 `json:"timeBeforeDeleteFromPresentInMillis,omitempty"`
}

// Validate validates this edit backup params
func (m *EditBackupParams) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateExpiryTimeUnit(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateStorageConfigUUID(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

var editBackupParamsTypeExpiryTimeUnitPropEnum []interface{}

func init() {
	var res []string
	if err := json.Unmarshal([]byte(`["NANOSECONDS","MICROSECONDS","MILLISECONDS","SECONDS","MINUTES","HOURS","DAYS","MONTHS","YEARS"]`), &res); err != nil {
		panic(err)
	}
	for _, v := range res {
		editBackupParamsTypeExpiryTimeUnitPropEnum = append(editBackupParamsTypeExpiryTimeUnitPropEnum, v)
	}
}

const (

	// EditBackupParamsExpiryTimeUnitNANOSECONDS captures enum value "NANOSECONDS"
	EditBackupParamsExpiryTimeUnitNANOSECONDS string = "NANOSECONDS"

	// EditBackupParamsExpiryTimeUnitMICROSECONDS captures enum value "MICROSECONDS"
	EditBackupParamsExpiryTimeUnitMICROSECONDS string = "MICROSECONDS"

	// EditBackupParamsExpiryTimeUnitMILLISECONDS captures enum value "MILLISECONDS"
	EditBackupParamsExpiryTimeUnitMILLISECONDS string = "MILLISECONDS"

	// EditBackupParamsExpiryTimeUnitSECONDS captures enum value "SECONDS"
	EditBackupParamsExpiryTimeUnitSECONDS string = "SECONDS"

	// EditBackupParamsExpiryTimeUnitMINUTES captures enum value "MINUTES"
	EditBackupParamsExpiryTimeUnitMINUTES string = "MINUTES"

	// EditBackupParamsExpiryTimeUnitHOURS captures enum value "HOURS"
	EditBackupParamsExpiryTimeUnitHOURS string = "HOURS"

	// EditBackupParamsExpiryTimeUnitDAYS captures enum value "DAYS"
	EditBackupParamsExpiryTimeUnitDAYS string = "DAYS"

	// EditBackupParamsExpiryTimeUnitMONTHS captures enum value "MONTHS"
	EditBackupParamsExpiryTimeUnitMONTHS string = "MONTHS"

	// EditBackupParamsExpiryTimeUnitYEARS captures enum value "YEARS"
	EditBackupParamsExpiryTimeUnitYEARS string = "YEARS"
)

// prop value enum
func (m *EditBackupParams) validateExpiryTimeUnitEnum(path, location string, value string) error {
	if err := validate.EnumCase(path, location, value, editBackupParamsTypeExpiryTimeUnitPropEnum, true); err != nil {
		return err
	}
	return nil
}

func (m *EditBackupParams) validateExpiryTimeUnit(formats strfmt.Registry) error {
	if swag.IsZero(m.ExpiryTimeUnit) { // not required
		return nil
	}

	// value enum
	if err := m.validateExpiryTimeUnitEnum("expiryTimeUnit", "body", m.ExpiryTimeUnit); err != nil {
		return err
	}

	return nil
}

func (m *EditBackupParams) validateStorageConfigUUID(formats strfmt.Registry) error {
	if swag.IsZero(m.StorageConfigUUID) { // not required
		return nil
	}

	if err := validate.FormatOf("storageConfigUUID", "body", "uuid", m.StorageConfigUUID.String(), formats); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this edit backup params based on context it is used
func (m *EditBackupParams) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *EditBackupParams) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *EditBackupParams) UnmarshalBinary(b []byte) error {
	var res EditBackupParams
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}