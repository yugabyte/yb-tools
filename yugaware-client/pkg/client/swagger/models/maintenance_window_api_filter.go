// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// MaintenanceWindowAPIFilter maintenance window Api filter
//
// swagger:model MaintenanceWindowApiFilter
type MaintenanceWindowAPIFilter struct {

	// states
	// Required: true
	// Unique: true
	States []string `json:"states"`

	// uuids
	// Required: true
	// Unique: true
	Uuids []strfmt.UUID `json:"uuids"`
}

// Validate validates this maintenance window Api filter
func (m *MaintenanceWindowAPIFilter) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateStates(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateUuids(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

var maintenanceWindowApiFilterStatesItemsEnum []interface{}

func init() {
	var res []string
	if err := json.Unmarshal([]byte(`["FINISHED","ACTIVE","PENDING"]`), &res); err != nil {
		panic(err)
	}
	for _, v := range res {
		maintenanceWindowApiFilterStatesItemsEnum = append(maintenanceWindowApiFilterStatesItemsEnum, v)
	}
}

func (m *MaintenanceWindowAPIFilter) validateStatesItemsEnum(path, location string, value string) error {
	if err := validate.EnumCase(path, location, value, maintenanceWindowApiFilterStatesItemsEnum, true); err != nil {
		return err
	}
	return nil
}

func (m *MaintenanceWindowAPIFilter) validateStates(formats strfmt.Registry) error {

	if err := validate.Required("states", "body", m.States); err != nil {
		return err
	}

	if err := validate.UniqueItems("states", "body", m.States); err != nil {
		return err
	}

	for i := 0; i < len(m.States); i++ {

		// value enum
		if err := m.validateStatesItemsEnum("states"+"."+strconv.Itoa(i), "body", m.States[i]); err != nil {
			return err
		}

	}

	return nil
}

func (m *MaintenanceWindowAPIFilter) validateUuids(formats strfmt.Registry) error {

	if err := validate.Required("uuids", "body", m.Uuids); err != nil {
		return err
	}

	if err := validate.UniqueItems("uuids", "body", m.Uuids); err != nil {
		return err
	}

	for i := 0; i < len(m.Uuids); i++ {

		if err := validate.FormatOf("uuids"+"."+strconv.Itoa(i), "body", "uuid", m.Uuids[i].String(), formats); err != nil {
			return err
		}

	}

	return nil
}

// ContextValidate validates this maintenance window Api filter based on context it is used
func (m *MaintenanceWindowAPIFilter) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *MaintenanceWindowAPIFilter) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *MaintenanceWindowAPIFilter) UnmarshalBinary(b []byte) error {
	var res MaintenanceWindowAPIFilter
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
