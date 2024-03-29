// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// ExtraDependencies Extra dependencies
//
// swagger:model ExtraDependencies
type ExtraDependencies struct {

	// Install node exporter on nodes
	InstallNodeExporter bool `json:"installNodeExporter,omitempty"`
}

// Validate validates this extra dependencies
func (m *ExtraDependencies) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this extra dependencies based on context it is used
func (m *ExtraDependencies) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *ExtraDependencies) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *ExtraDependencies) UnmarshalBinary(b []byte) error {
	var res ExtraDependencies
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
