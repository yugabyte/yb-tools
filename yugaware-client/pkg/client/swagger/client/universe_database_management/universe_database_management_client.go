// Code generated by go-swagger; DO NOT EDIT.

package universe_database_management

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// New creates a new universe database management API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) ClientService {
	return &Client{transport: transport, formats: formats}
}

/*
Client for universe database management API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

// ClientOption is the option for Client methods
type ClientOption func(*runtime.ClientOperation)

// ClientService is the interface for Client methods
type ClientService interface {
	CreateUserInDB(params *CreateUserInDBParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*CreateUserInDBOK, error)

	RunInShell(params *RunInShellParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*RunInShellOK, error)

	RunYsqlQueryUniverse(params *RunYsqlQueryUniverseParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*RunYsqlQueryUniverseOK, error)

	SetDatabaseCredentials(params *SetDatabaseCredentialsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*SetDatabaseCredentialsOK, error)

	SetTransport(transport runtime.ClientTransport)
}

/*
  CreateUserInDB creates a database user for a universe
*/
func (a *Client) CreateUserInDB(params *CreateUserInDBParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*CreateUserInDBOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewCreateUserInDBParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "createUserInDB",
		Method:             "POST",
		PathPattern:        "/api/v1/customers/{cUUID}/universes/{uniUUID}/create_db_credentials",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &CreateUserInDBReader{formats: a.formats},
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
	success, ok := result.(*CreateUserInDBOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for createUserInDB: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  RunInShell runs a shell command

  This operation is no longer supported, for security reasons.
*/
func (a *Client) RunInShell(params *RunInShellParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*RunInShellOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewRunInShellParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "runInShell",
		Method:             "POST",
		PathPattern:        "/api/v1/customers/{cUUID}/universes/{uniUUID}/run_in_shell",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &RunInShellReader{formats: a.formats},
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
	success, ok := result.(*RunInShellOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for runInShell: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  RunYsqlQueryUniverse runs a y SQL query in a universe

  Runs a YSQL query. Only valid when the platform is running in `OSS` mode.
*/
func (a *Client) RunYsqlQueryUniverse(params *RunYsqlQueryUniverseParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*RunYsqlQueryUniverseOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewRunYsqlQueryUniverseParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "runYsqlQueryUniverse",
		Method:             "POST",
		PathPattern:        "/api/v1/customers/{cUUID}/universes/{uniUUID}/run_query",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &RunYsqlQueryUniverseReader{formats: a.formats},
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
	success, ok := result.(*RunYsqlQueryUniverseOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for runYsqlQueryUniverse: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  SetDatabaseCredentials sets a universe s database credentials
*/
func (a *Client) SetDatabaseCredentials(params *SetDatabaseCredentialsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*SetDatabaseCredentialsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewSetDatabaseCredentialsParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "setDatabaseCredentials",
		Method:             "POST",
		PathPattern:        "/api/v1/customers/{cUUID}/universes/{uniUUID}/update_db_credentials",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &SetDatabaseCredentialsReader{formats: a.formats},
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
	success, ok := result.(*SetDatabaseCredentialsOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for setDatabaseCredentials: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
