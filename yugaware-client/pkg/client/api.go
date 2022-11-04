package client

import (
	"strings"

	"github.com/go-openapi/strfmt"
	"github.com/yugabyte/yb-tools/yugaware-client/entity/yugaware"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/certificate_info"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/cloud_providers"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/customer_configuration"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/universe_management"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
)

func (c *YugawareClient) RegisterYugaware(request *yugaware.RegisterYugawareRequest) (*yugaware.RegisterYugawareResponse, error) {

	RegisterPath := "/api/v1/register"
	response := &yugaware.RegisterYugawareResponse{}

	_, err := c.newRequest().
		Post().
		Path(RegisterPath).
		RequestBody(request).
		DecodeResponseInto(response).
		Do()

	if err != nil {
		//if yr.Response.StatusCode == 401 {
		//	return nil,
		//}
		return nil, err
	}

	return response, nil
}

func (c *YugawareClient) Login(request *yugaware.LoginRequest) (*yugaware.LoginResponse, error) {

	LoginPath := "/api/v1/login"
	response := &yugaware.LoginResponse{}

	_, err := c.newRequest().
		Post().
		Path(LoginPath).
		RequestBody(request).
		DecodeResponseInto(response).
		Do()

	if err != nil {
		return nil, err
	}

	err = c.persistCookies()
	if err != nil {
		return nil, err
	}

	c.authToken = response.AuthToken
	c.customerUUID = strfmt.UUID(response.CustomerUUID)
	c.userUUID = strfmt.UUID(response.UserUUID)

	return response, nil
}

func (c *YugawareClient) Logout() error {
	c.expireSession()

	return c.cookiejar.Save()
}

func (c *YugawareClient) GetUniverseByIdentifier(identifier string) (*models.UniverseResp, error) {

	params := universe_management.NewListUniversesParams().
		WithCUUID(c.CustomerUUID())

	universes, err := c.PlatformAPIs.UniverseManagement.ListUniverses(params, c.SwaggerAuth)
	if err != nil {
		return nil, err
	}

	for _, universe := range universes.GetPayload() {
		if universe.Name == identifier {
			return universe, nil
		}

		if universe.UniverseUUID == strfmt.UUID(strings.ToLower(identifier)) {
			return universe, nil
		}
	}

	return nil, nil
}

func (c *YugawareClient) GetCertByIdentifier(identifier string) (*models.CertificateInfo, error) {

	params := certificate_info.NewGetListOfCertificateParams().WithCUUID(c.CustomerUUID())

	certs, err := c.PlatformAPIs.CertificateInfo.GetListOfCertificate(params, c.SwaggerAuth)
	if err != nil {
		return nil, err
	}

	for _, cert := range certs.GetPayload() {
		if cert.Label == identifier {
			return cert, nil
		}

		if cert.UUID == strfmt.UUID(strings.ToLower(identifier)) {
			return cert, nil
		}
	}

	return nil, nil
}

func (c *YugawareClient) GetStorageConfigByIdentifier(identifier string) (*models.CustomerConfigUI, error) {
	configParams := customer_configuration.NewGetListOfCustomerConfigParams().
		WithCUUID(c.CustomerUUID())

	configs, err := c.PlatformAPIs.CustomerConfiguration.GetListOfCustomerConfig(configParams, c.SwaggerAuth)
	if err != nil {
		return nil, err
	}

	for _, config := range configs.GetPayload() {
		if *config.Type == models.CustomerConfigTypeSTORAGE {
			if *config.ConfigName == identifier {
				return config, nil
			}
			if config.ConfigUUID == strfmt.UUID(identifier) {
				return config, nil
			}
		}
	}

	return nil, nil
}

func (c *YugawareClient) GetProviderByIdentifier(identifier string) (*models.Provider, error) {
	params := cloud_providers.NewGetListOfProvidersParams().
		WithCUUID(c.CustomerUUID())
	providers, err := c.PlatformAPIs.CloudProviders.GetListOfProviders(params, c.SwaggerAuth)
	if err != nil {
		return nil, err
	}

	for _, provider := range providers.GetPayload() {
		if provider.Name == identifier {
			return provider, nil
		}
		if provider.UUID == strfmt.UUID(identifier) {
			return provider, nil
		}
	}

	return nil, nil
}
