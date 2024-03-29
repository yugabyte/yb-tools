// Code generated by go-swagger; DO NOT EDIT.

package customer_configuration

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// New creates a new customer configuration API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) ClientService {
	return &Client{transport: transport, formats: formats}
}

/*
Client for customer configuration API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

// ClientOption is the option for Client methods
type ClientOption func(*runtime.ClientOperation)

// ClientService is the interface for Client methods
type ClientService interface {
	CreateCustomerConfig(params *CreateCustomerConfigParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*CreateCustomerConfigOK, error)

	DeleteCustomerConfig(params *DeleteCustomerConfigParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*DeleteCustomerConfigOK, error)

	DeleteCustomerConfigV2(params *DeleteCustomerConfigV2Params, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*DeleteCustomerConfigV2OK, error)

	EditCustomerConfig(params *EditCustomerConfigParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*EditCustomerConfigOK, error)

	EditCustomerConfigV2(params *EditCustomerConfigV2Params, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*EditCustomerConfigV2OK, error)

	GenerateAPIToken(params *GenerateAPITokenParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GenerateAPITokenOK, error)

	GetListOfCustomerConfig(params *GetListOfCustomerConfigParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetListOfCustomerConfigOK, error)

	SetTransport(transport runtime.ClientTransport)
}

/*
  CreateCustomerConfig creates a customer configuration
*/
func (a *Client) CreateCustomerConfig(params *CreateCustomerConfigParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*CreateCustomerConfigOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewCreateCustomerConfigParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "createCustomerConfig",
		Method:             "POST",
		PathPattern:        "/api/v1/customers/{cUUID}/configs",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &CreateCustomerConfigReader{formats: a.formats},
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
	success, ok := result.(*CreateCustomerConfigOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for createCustomerConfig: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  DeleteCustomerConfig deletes a customer configuration
*/
func (a *Client) DeleteCustomerConfig(params *DeleteCustomerConfigParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*DeleteCustomerConfigOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewDeleteCustomerConfigParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "deleteCustomerConfig",
		Method:             "DELETE",
		PathPattern:        "/api/v1/customers/{cUUID}/configs/{configUUID}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &DeleteCustomerConfigReader{formats: a.formats},
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
	success, ok := result.(*DeleteCustomerConfigOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for deleteCustomerConfig: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  DeleteCustomerConfigV2 deletes a customer configuration v2
*/
func (a *Client) DeleteCustomerConfigV2(params *DeleteCustomerConfigV2Params, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*DeleteCustomerConfigV2OK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewDeleteCustomerConfigV2Params()
	}
	op := &runtime.ClientOperation{
		ID:                 "deleteCustomerConfigV2",
		Method:             "DELETE",
		PathPattern:        "/api/v1/customers/{cUUID}/configs/{configUUID}/delete",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &DeleteCustomerConfigV2Reader{formats: a.formats},
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
	success, ok := result.(*DeleteCustomerConfigV2OK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for deleteCustomerConfigV2: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  EditCustomerConfig updates a customer configuration
*/
func (a *Client) EditCustomerConfig(params *EditCustomerConfigParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*EditCustomerConfigOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewEditCustomerConfigParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "editCustomerConfig",
		Method:             "PUT",
		PathPattern:        "/api/v1/customers/{cUUID}/configs/{configUUID}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &EditCustomerConfigReader{formats: a.formats},
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
	success, ok := result.(*EditCustomerConfigOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for editCustomerConfig: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  EditCustomerConfigV2 updates a customer configuration v2

  Changes from upstream: This method is called 'editCustomerConfig' in the upstream swagger.json
*/
func (a *Client) EditCustomerConfigV2(params *EditCustomerConfigV2Params, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*EditCustomerConfigV2OK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewEditCustomerConfigV2Params()
	}
	op := &runtime.ClientOperation{
		ID:                 "editCustomerConfigV2",
		Method:             "PUT",
		PathPattern:        "/api/v1/customers/{cUUID}/configs/{configUUID}/edit",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &EditCustomerConfigV2Reader{formats: a.formats},
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
	success, ok := result.(*EditCustomerConfigV2OK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for editCustomerConfigV2: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  GenerateAPIToken generates an API token for the current user

  UNOFFICIAL API ADDITION - Requires a DUMMY body to work around issue https://yugabyte.atlassian.net/browse/PLAT-2076
*/
func (a *Client) GenerateAPIToken(params *GenerateAPITokenParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GenerateAPITokenOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGenerateAPITokenParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "generateAPIToken",
		Method:             "PUT",
		PathPattern:        "/api/v1/customers/{cUUID}/api_token",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &GenerateAPITokenReader{formats: a.formats},
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
	success, ok := result.(*GenerateAPITokenOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for generateAPIToken: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  GetListOfCustomerConfig lists all customer configurations
*/
func (a *Client) GetListOfCustomerConfig(params *GetListOfCustomerConfigParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetListOfCustomerConfigOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetListOfCustomerConfigParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "getListOfCustomerConfig",
		Method:             "GET",
		PathPattern:        "/api/v1/customers/{cUUID}/configs",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &GetListOfCustomerConfigReader{formats: a.formats},
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
	success, ok := result.(*GetListOfCustomerConfigOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for getListOfCustomerConfig: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
