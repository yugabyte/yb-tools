// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// AccessKeyID access key Id
//
// swagger:model AccessKeyId
type AccessKeyID struct {

	// key code
	KeyCode string `json:"keyCode,omitempty"`

	// provider UUID
	// Format: uuid
	ProviderUUID strfmt.UUID `json:"providerUUID,omitempty"`
}

// Validate validates this access key Id
func (m *AccessKeyID) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateProviderUUID(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *AccessKeyID) validateProviderUUID(formats strfmt.Registry) error {
	if swag.IsZero(m.ProviderUUID) { // not required
		return nil
	}

	if err := validate.FormatOf("providerUUID", "body", "uuid", m.ProviderUUID.String(), formats); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this access key Id based on context it is used
func (m *AccessKeyID) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *AccessKeyID) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *AccessKeyID) UnmarshalBinary(b []byte) error {
	var res AccessKeyID
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
