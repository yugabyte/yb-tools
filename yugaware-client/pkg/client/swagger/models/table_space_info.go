// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"strconv"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// TableSpaceInfo Tablespace information
//
// swagger:model TableSpaceInfo
type TableSpaceInfo struct {

	// Tablespace Name
	// Required: true
	// Max Length: 2147483647
	// Min Length: 1
	Name *string `json:"name"`

	// numReplicas
	// Minimum: 1
	NumReplicas int32 `json:"numReplicas,omitempty"`

	// placements
	// Required: true
	// Max Items: 2147483647
	// Min Items: 1
	PlacementBlocks []*PlacementBlock `json:"placementBlocks"`
}

// Validate validates this table space info
func (m *TableSpaceInfo) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateName(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateNumReplicas(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validatePlacementBlocks(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *TableSpaceInfo) validateName(formats strfmt.Registry) error {

	if err := validate.Required("name", "body", m.Name); err != nil {
		return err
	}

	if err := validate.MinLength("name", "body", *m.Name, 1); err != nil {
		return err
	}

	if err := validate.MaxLength("name", "body", *m.Name, 2147483647); err != nil {
		return err
	}

	return nil
}

func (m *TableSpaceInfo) validateNumReplicas(formats strfmt.Registry) error {
	if swag.IsZero(m.NumReplicas) { // not required
		return nil
	}

	if err := validate.MinimumInt("numReplicas", "body", int64(m.NumReplicas), 1, false); err != nil {
		return err
	}

	return nil
}

func (m *TableSpaceInfo) validatePlacementBlocks(formats strfmt.Registry) error {

	if err := validate.Required("placementBlocks", "body", m.PlacementBlocks); err != nil {
		return err
	}

	iPlacementBlocksSize := int64(len(m.PlacementBlocks))

	if err := validate.MinItems("placementBlocks", "body", iPlacementBlocksSize, 1); err != nil {
		return err
	}

	if err := validate.MaxItems("placementBlocks", "body", iPlacementBlocksSize, 2147483647); err != nil {
		return err
	}

	for i := 0; i < len(m.PlacementBlocks); i++ {
		if swag.IsZero(m.PlacementBlocks[i]) { // not required
			continue
		}

		if m.PlacementBlocks[i] != nil {
			if err := m.PlacementBlocks[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("placementBlocks" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("placementBlocks" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// ContextValidate validate this table space info based on the context it is used
func (m *TableSpaceInfo) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidatePlacementBlocks(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *TableSpaceInfo) contextValidatePlacementBlocks(ctx context.Context, formats strfmt.Registry) error {

	for i := 0; i < len(m.PlacementBlocks); i++ {

		if m.PlacementBlocks[i] != nil {
			if err := m.PlacementBlocks[i].ContextValidate(ctx, formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("placementBlocks" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("placementBlocks" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// MarshalBinary interface implementation
func (m *TableSpaceInfo) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *TableSpaceInfo) UnmarshalBinary(b []byte) error {
	var res TableSpaceInfo
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
