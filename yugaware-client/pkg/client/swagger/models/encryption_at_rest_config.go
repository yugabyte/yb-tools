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

// EncryptionAtRestConfig Encryption at rest configuration
//
// swagger:model EncryptionAtRestConfig
type EncryptionAtRestConfig struct {

	// Whether a universe is currently encrypted at rest
	EncryptionAtRestEnabled bool `json:"encryptionAtRestEnabled,omitempty"`

	// KMS configuration UUID
	// Format: uuid
	KmsConfigUUID strfmt.UUID `json:"kmsConfigUUID,omitempty"`

	// Operation type: enable, disable, or rotate the universe key/encryption at rest
	// Enum: [ENABLE DISABLE UNDEFINED]
	OpType string `json:"opType,omitempty"`

	// Whether to generate a data key or just retrieve the CMK ARN
	// Enum: [CMK DATA_KEY]
	Type string `json:"type,omitempty"`
}

// Validate validates this encryption at rest config
func (m *EncryptionAtRestConfig) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateKmsConfigUUID(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateOpType(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateType(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *EncryptionAtRestConfig) validateKmsConfigUUID(formats strfmt.Registry) error {
	if swag.IsZero(m.KmsConfigUUID) { // not required
		return nil
	}

	if err := validate.FormatOf("kmsConfigUUID", "body", "uuid", m.KmsConfigUUID.String(), formats); err != nil {
		return err
	}

	return nil
}

var encryptionAtRestConfigTypeOpTypePropEnum []interface{}

func init() {
	var res []string
	if err := json.Unmarshal([]byte(`["ENABLE","DISABLE","UNDEFINED"]`), &res); err != nil {
		panic(err)
	}
	for _, v := range res {
		encryptionAtRestConfigTypeOpTypePropEnum = append(encryptionAtRestConfigTypeOpTypePropEnum, v)
	}
}

const (

	// EncryptionAtRestConfigOpTypeENABLE captures enum value "ENABLE"
	EncryptionAtRestConfigOpTypeENABLE string = "ENABLE"

	// EncryptionAtRestConfigOpTypeDISABLE captures enum value "DISABLE"
	EncryptionAtRestConfigOpTypeDISABLE string = "DISABLE"

	// EncryptionAtRestConfigOpTypeUNDEFINED captures enum value "UNDEFINED"
	EncryptionAtRestConfigOpTypeUNDEFINED string = "UNDEFINED"
)

// prop value enum
func (m *EncryptionAtRestConfig) validateOpTypeEnum(path, location string, value string) error {
	if err := validate.EnumCase(path, location, value, encryptionAtRestConfigTypeOpTypePropEnum, true); err != nil {
		return err
	}
	return nil
}

func (m *EncryptionAtRestConfig) validateOpType(formats strfmt.Registry) error {
	if swag.IsZero(m.OpType) { // not required
		return nil
	}

	// value enum
	if err := m.validateOpTypeEnum("opType", "body", m.OpType); err != nil {
		return err
	}

	return nil
}

var encryptionAtRestConfigTypeTypePropEnum []interface{}

func init() {
	var res []string
	if err := json.Unmarshal([]byte(`["CMK","DATA_KEY"]`), &res); err != nil {
		panic(err)
	}
	for _, v := range res {
		encryptionAtRestConfigTypeTypePropEnum = append(encryptionAtRestConfigTypeTypePropEnum, v)
	}
}

const (

	// EncryptionAtRestConfigTypeCMK captures enum value "CMK"
	EncryptionAtRestConfigTypeCMK string = "CMK"

	// EncryptionAtRestConfigTypeDATAKEY captures enum value "DATA_KEY"
	EncryptionAtRestConfigTypeDATAKEY string = "DATA_KEY"
)

// prop value enum
func (m *EncryptionAtRestConfig) validateTypeEnum(path, location string, value string) error {
	if err := validate.EnumCase(path, location, value, encryptionAtRestConfigTypeTypePropEnum, true); err != nil {
		return err
	}
	return nil
}

func (m *EncryptionAtRestConfig) validateType(formats strfmt.Registry) error {
	if swag.IsZero(m.Type) { // not required
		return nil
	}

	// value enum
	if err := m.validateTypeEnum("type", "body", m.Type); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this encryption at rest config based on context it is used
func (m *EncryptionAtRestConfig) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *EncryptionAtRestConfig) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *EncryptionAtRestConfig) UnmarshalBinary(b []byte) error {
	var res EncryptionAtRestConfig
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
