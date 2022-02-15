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

// PlatformLoggingConfig platform logging config
//
// swagger:model PlatformLoggingConfig
type PlatformLoggingConfig struct {

	// level
	// Required: true
	// Enum: [ERROR WARN INFO DEBUG TRACE]
	Level *string `json:"level"`

	// max history
	// Required: true
	// Minimum: 0
	MaxHistory *int32 `json:"maxHistory"`

	// rollover pattern
	// Required: true
	RolloverPattern *string `json:"rolloverPattern"`
}

// Validate validates this platform logging config
func (m *PlatformLoggingConfig) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateLevel(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateMaxHistory(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateRolloverPattern(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

var platformLoggingConfigTypeLevelPropEnum []interface{}

func init() {
	var res []string
	if err := json.Unmarshal([]byte(`["ERROR","WARN","INFO","DEBUG","TRACE"]`), &res); err != nil {
		panic(err)
	}
	for _, v := range res {
		platformLoggingConfigTypeLevelPropEnum = append(platformLoggingConfigTypeLevelPropEnum, v)
	}
}

const (

	// PlatformLoggingConfigLevelERROR captures enum value "ERROR"
	PlatformLoggingConfigLevelERROR string = "ERROR"

	// PlatformLoggingConfigLevelWARN captures enum value "WARN"
	PlatformLoggingConfigLevelWARN string = "WARN"

	// PlatformLoggingConfigLevelINFO captures enum value "INFO"
	PlatformLoggingConfigLevelINFO string = "INFO"

	// PlatformLoggingConfigLevelDEBUG captures enum value "DEBUG"
	PlatformLoggingConfigLevelDEBUG string = "DEBUG"

	// PlatformLoggingConfigLevelTRACE captures enum value "TRACE"
	PlatformLoggingConfigLevelTRACE string = "TRACE"
)

// prop value enum
func (m *PlatformLoggingConfig) validateLevelEnum(path, location string, value string) error {
	if err := validate.EnumCase(path, location, value, platformLoggingConfigTypeLevelPropEnum, true); err != nil {
		return err
	}
	return nil
}

func (m *PlatformLoggingConfig) validateLevel(formats strfmt.Registry) error {

	if err := validate.Required("level", "body", m.Level); err != nil {
		return err
	}

	// value enum
	if err := m.validateLevelEnum("level", "body", *m.Level); err != nil {
		return err
	}

	return nil
}

func (m *PlatformLoggingConfig) validateMaxHistory(formats strfmt.Registry) error {

	if err := validate.Required("maxHistory", "body", m.MaxHistory); err != nil {
		return err
	}

	if err := validate.MinimumInt("maxHistory", "body", int64(*m.MaxHistory), 0, false); err != nil {
		return err
	}

	return nil
}

func (m *PlatformLoggingConfig) validateRolloverPattern(formats strfmt.Registry) error {

	if err := validate.Required("rolloverPattern", "body", m.RolloverPattern); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this platform logging config based on context it is used
func (m *PlatformLoggingConfig) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *PlatformLoggingConfig) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *PlatformLoggingConfig) UnmarshalBinary(b []byte) error {
	var res PlatformLoggingConfig
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
