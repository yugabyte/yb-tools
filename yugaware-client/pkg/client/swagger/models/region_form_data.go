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

// RegionFormData region form data
//
// swagger:model RegionFormData
type RegionFormData struct {

	// code
	// Required: true
	Code *string `json:"code"`

	// dest vpc Id
	// Required: true
	DestVpcID *string `json:"destVpcId"`

	// host vpc Id
	// Required: true
	HostVpcID *string `json:"hostVpcId"`

	// host vpc region
	// Required: true
	HostVpcRegion *string `json:"hostVpcRegion"`

	// latitude
	// Required: true
	Latitude *float64 `json:"latitude"`

	// longitude
	// Required: true
	Longitude *float64 `json:"longitude"`

	// name
	// Required: true
	Name *string `json:"name"`

	// yb image
	// Required: true
	YbImage *string `json:"ybImage"`
}

// Validate validates this region form data
func (m *RegionFormData) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateCode(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateDestVpcID(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateHostVpcID(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateHostVpcRegion(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateLatitude(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateLongitude(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateName(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateYbImage(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *RegionFormData) validateCode(formats strfmt.Registry) error {

	if err := validate.Required("code", "body", m.Code); err != nil {
		return err
	}

	return nil
}

func (m *RegionFormData) validateDestVpcID(formats strfmt.Registry) error {

	if err := validate.Required("destVpcId", "body", m.DestVpcID); err != nil {
		return err
	}

	return nil
}

func (m *RegionFormData) validateHostVpcID(formats strfmt.Registry) error {

	if err := validate.Required("hostVpcId", "body", m.HostVpcID); err != nil {
		return err
	}

	return nil
}

func (m *RegionFormData) validateHostVpcRegion(formats strfmt.Registry) error {

	if err := validate.Required("hostVpcRegion", "body", m.HostVpcRegion); err != nil {
		return err
	}

	return nil
}

func (m *RegionFormData) validateLatitude(formats strfmt.Registry) error {

	if err := validate.Required("latitude", "body", m.Latitude); err != nil {
		return err
	}

	return nil
}

func (m *RegionFormData) validateLongitude(formats strfmt.Registry) error {

	if err := validate.Required("longitude", "body", m.Longitude); err != nil {
		return err
	}

	return nil
}

func (m *RegionFormData) validateName(formats strfmt.Registry) error {

	if err := validate.Required("name", "body", m.Name); err != nil {
		return err
	}

	return nil
}

func (m *RegionFormData) validateYbImage(formats strfmt.Registry) error {

	if err := validate.Required("ybImage", "body", m.YbImage); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this region form data based on context it is used
func (m *RegionFormData) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *RegionFormData) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *RegionFormData) UnmarshalBinary(b []byte) error {
	var res RegionFormData
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
