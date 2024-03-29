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

// MaintenanceWindowPagedAPIQuery maintenance window paged Api query
//
// swagger:model MaintenanceWindowPagedApiQuery
type MaintenanceWindowPagedAPIQuery struct {

	// direction
	// Required: true
	// Enum: [ASC DESC]
	Direction *string `json:"direction"`

	// filter
	// Required: true
	Filter *MaintenanceWindowAPIFilter `json:"filter"`

	// limit
	// Required: true
	Limit *int32 `json:"limit"`

	// need total count
	// Required: true
	NeedTotalCount *bool `json:"needTotalCount"`

	// offset
	// Required: true
	Offset *int32 `json:"offset"`

	// sort by
	// Required: true
	// Enum: [uuid name createTime startTime endTime state]
	SortBy *string `json:"sortBy"`
}

// Validate validates this maintenance window paged Api query
func (m *MaintenanceWindowPagedAPIQuery) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateDirection(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateFilter(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateLimit(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateNeedTotalCount(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateOffset(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateSortBy(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

var maintenanceWindowPagedApiQueryTypeDirectionPropEnum []interface{}

func init() {
	var res []string
	if err := json.Unmarshal([]byte(`["ASC","DESC"]`), &res); err != nil {
		panic(err)
	}
	for _, v := range res {
		maintenanceWindowPagedApiQueryTypeDirectionPropEnum = append(maintenanceWindowPagedApiQueryTypeDirectionPropEnum, v)
	}
}

const (

	// MaintenanceWindowPagedAPIQueryDirectionASC captures enum value "ASC"
	MaintenanceWindowPagedAPIQueryDirectionASC string = "ASC"

	// MaintenanceWindowPagedAPIQueryDirectionDESC captures enum value "DESC"
	MaintenanceWindowPagedAPIQueryDirectionDESC string = "DESC"
)

// prop value enum
func (m *MaintenanceWindowPagedAPIQuery) validateDirectionEnum(path, location string, value string) error {
	if err := validate.EnumCase(path, location, value, maintenanceWindowPagedApiQueryTypeDirectionPropEnum, true); err != nil {
		return err
	}
	return nil
}

func (m *MaintenanceWindowPagedAPIQuery) validateDirection(formats strfmt.Registry) error {

	if err := validate.Required("direction", "body", m.Direction); err != nil {
		return err
	}

	// value enum
	if err := m.validateDirectionEnum("direction", "body", *m.Direction); err != nil {
		return err
	}

	return nil
}

func (m *MaintenanceWindowPagedAPIQuery) validateFilter(formats strfmt.Registry) error {

	if err := validate.Required("filter", "body", m.Filter); err != nil {
		return err
	}

	if m.Filter != nil {
		if err := m.Filter.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("filter")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("filter")
			}
			return err
		}
	}

	return nil
}

func (m *MaintenanceWindowPagedAPIQuery) validateLimit(formats strfmt.Registry) error {

	if err := validate.Required("limit", "body", m.Limit); err != nil {
		return err
	}

	return nil
}

func (m *MaintenanceWindowPagedAPIQuery) validateNeedTotalCount(formats strfmt.Registry) error {

	if err := validate.Required("needTotalCount", "body", m.NeedTotalCount); err != nil {
		return err
	}

	return nil
}

func (m *MaintenanceWindowPagedAPIQuery) validateOffset(formats strfmt.Registry) error {

	if err := validate.Required("offset", "body", m.Offset); err != nil {
		return err
	}

	return nil
}

var maintenanceWindowPagedApiQueryTypeSortByPropEnum []interface{}

func init() {
	var res []string
	if err := json.Unmarshal([]byte(`["uuid","name","createTime","startTime","endTime","state"]`), &res); err != nil {
		panic(err)
	}
	for _, v := range res {
		maintenanceWindowPagedApiQueryTypeSortByPropEnum = append(maintenanceWindowPagedApiQueryTypeSortByPropEnum, v)
	}
}

const (

	// MaintenanceWindowPagedAPIQuerySortByUUID captures enum value "uuid"
	MaintenanceWindowPagedAPIQuerySortByUUID string = "uuid"

	// MaintenanceWindowPagedAPIQuerySortByName captures enum value "name"
	MaintenanceWindowPagedAPIQuerySortByName string = "name"

	// MaintenanceWindowPagedAPIQuerySortByCreateTime captures enum value "createTime"
	MaintenanceWindowPagedAPIQuerySortByCreateTime string = "createTime"

	// MaintenanceWindowPagedAPIQuerySortByStartTime captures enum value "startTime"
	MaintenanceWindowPagedAPIQuerySortByStartTime string = "startTime"

	// MaintenanceWindowPagedAPIQuerySortByEndTime captures enum value "endTime"
	MaintenanceWindowPagedAPIQuerySortByEndTime string = "endTime"

	// MaintenanceWindowPagedAPIQuerySortByState captures enum value "state"
	MaintenanceWindowPagedAPIQuerySortByState string = "state"
)

// prop value enum
func (m *MaintenanceWindowPagedAPIQuery) validateSortByEnum(path, location string, value string) error {
	if err := validate.EnumCase(path, location, value, maintenanceWindowPagedApiQueryTypeSortByPropEnum, true); err != nil {
		return err
	}
	return nil
}

func (m *MaintenanceWindowPagedAPIQuery) validateSortBy(formats strfmt.Registry) error {

	if err := validate.Required("sortBy", "body", m.SortBy); err != nil {
		return err
	}

	// value enum
	if err := m.validateSortByEnum("sortBy", "body", *m.SortBy); err != nil {
		return err
	}

	return nil
}

// ContextValidate validate this maintenance window paged Api query based on the context it is used
func (m *MaintenanceWindowPagedAPIQuery) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateFilter(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *MaintenanceWindowPagedAPIQuery) contextValidateFilter(ctx context.Context, formats strfmt.Registry) error {

	if m.Filter != nil {
		if err := m.Filter.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("filter")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("filter")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *MaintenanceWindowPagedAPIQuery) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *MaintenanceWindowPagedAPIQuery) UnmarshalBinary(b []byte) error {
	var res MaintenanceWindowPagedAPIQuery
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
