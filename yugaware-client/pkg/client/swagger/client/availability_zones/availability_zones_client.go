// Code generated by go-swagger; DO NOT EDIT.

package availability_zones

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// New creates a new availability zones API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) ClientService {
	return &Client{transport: transport, formats: formats}
}

/*
Client for availability zones API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

// ClientOption is the option for Client methods
type ClientOption func(*runtime.ClientOperation)

// ClientService is the interface for Client methods
type ClientService interface {
	CreateAZ(params *CreateAZParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*CreateAZOK, error)

	DeleteAZ(params *DeleteAZParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*DeleteAZOK, error)

	ListOfAZ(params *ListOfAZParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListOfAZOK, error)

	SetTransport(transport runtime.ClientTransport)
}

/*
  CreateAZ creates an availability zone
*/
func (a *Client) CreateAZ(params *CreateAZParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*CreateAZOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewCreateAZParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "createAZ",
		Method:             "POST",
		PathPattern:        "/api/v1/customers/{cUUID}/providers/{pUUID}/regions/{rUUID}/zones",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &CreateAZReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*CreateAZOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for createAZ: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  DeleteAZ deletes an availability zone
*/
func (a *Client) DeleteAZ(params *DeleteAZParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*DeleteAZOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewDeleteAZParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "deleteAZ",
		Method:             "DELETE",
		PathPattern:        "/api/v1/customers/{cUUID}/providers/{pUUID}/regions/{rUUID}/zones/{azUUID}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &DeleteAZReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*DeleteAZOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for deleteAZ: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  ListOfAZ lists availability zones
*/
func (a *Client) ListOfAZ(params *ListOfAZParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListOfAZOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListOfAZParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "listOfAZ",
		Method:             "GET",
		PathPattern:        "/api/v1/customers/{cUUID}/providers/{pUUID}/regions/{rUUID}/zones",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &ListOfAZReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*ListOfAZOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for listOfAZ: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
