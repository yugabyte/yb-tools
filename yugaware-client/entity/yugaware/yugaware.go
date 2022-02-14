/*
Copyright © 2021 Yugabyte Support

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package yugaware

import (
	"github.com/yugabyte/yb-tools/pkg/util"
	"go.uber.org/zap/zapcore"
)

type AuthToken struct {
	AuthToken    string `json:"authToken"`
	CustomerUUID string `json:"customerUUID"`
	UserUUID     string `json:"userUUID"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *LoginRequest) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("email", r.Email)
	enc.AddString("password", util.MaskOut(r.Password))
	return nil
}

type LoginResponse AuthToken

type Provider struct {
	UUID           string      `json:"uuid"`
	Code           string      `json:"code"`
	Name           string      `json:"name"`
	Active         bool        `json:"active"`
	CustomerUUID   string      `json:"customerUUID"`
	HostedZoneID   string      `json:"hostedZoneId"`
	HostedZoneName string      `json:"hostedZoneName"`
	CloudParams    CloudParams `json:"cloudParams"`
	Config         interface{} `json:"config,omitempty"`
}

type Region struct {
	VpcID                 string `json:"vpcId"`
	VpcCidr               string `json:"vpcCidr"`
	AzToSubnetIds         string `json:"azToSubnetIds"`
	SubnetID              string `json:"subnetId"`
	CustomImageID         string `json:"customImageId"`
	CustomSecurityGroupID string `json:"customSecurityGroupId"`
}

type CloudParams struct {
	ErrorString          *string           `json:"errorString"`
	ProviderUUID         *string           `json:"providerUUID"`
	PerRegionMetadata    map[string]Region `json:"perRegionMetadata"`
	KeyPairName          interface{}       `json:"keyPairName"`
	SSHPrivateKeyContent interface{}       `json:"sshPrivateKeyContent"`
	SSHUser              interface{}       `json:"sshUser"`
	AirGapInstall        bool              `json:"airGapInstall"`
	SSHPort              int               `json:"sshPort"`
	HostVpcID            interface{}       `json:"hostVpcId"`
	HostVpcRegion        interface{}       `json:"hostVpcRegion"`
	CustomHostCidrs      []interface{}     `json:"customHostCidrs"`
	DestVpcID            interface{}       `json:"destVpcId"`
}

type GCPConfig struct {
	TokenURI                     string `json:"token_uri"`
	PrivateKeyID                 string `json:"private_key_id"`
	ClientX509CertURL            string `json:"client_x509_cert_url"`
	ProjectID                    string `json:"project_id"`
	AuthURI                      string `json:"auth_uri"`
	AuthProviderX509CertURL      string `json:"auth_provider_x509_cert_url"`
	ClientEmail                  string `json:"client_email"`
	GceHostProject               string `json:"GCE_HOST_PROJECT"`
	PrivateKey                   string `json:"private_key"`
	Type                         string `json:"type"`
	ClientID                     string `json:"client_id"`
	GceProject                   string `json:"GCE_PROJECT"`
	GceEmail                     string `json:"GCE_EMAIL"`
	GoogleApplicationCredentials string `json:"GOOGLE_APPLICATION_CREDENTIALS"`
	CustomGceNetwork             string `json:"CUSTOM_GCE_NETWORK"`
}

type KubernetesConfig struct {
	KubeconfigProvider            string `json:"KUBECONFIG_PROVIDER"`
	KubeconfigServiceAccount      string `json:"KUBECONFIG_SERVICE_ACCOUNT"`
	KubeconfigImageRegistry       string `json:"KUBECONFIG_IMAGE_REGISTRY"`
	KubeconfigImagePullSecretName string `json:"KUBECONFIG_IMAGE_PULL_SECRET_NAME"`
	KubeconfigPullSecretName      string `json:"KUBECONFIG_PULL_SECRET_NAME"`
	KubeconfigPullSecretContent   string `json:"KUBECONFIG_PULL_SECRET_CONTENT"`
}

func (r *KubernetesConfig) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("KUBECONFIG_PROVIDER", r.KubeconfigProvider)
	enc.AddString("KUBECONFIG_SERVICE_ACCOUNT", r.KubeconfigServiceAccount)
	enc.AddString("KUBECONFIG_IMAGE_REGISTRY", r.KubeconfigImageRegistry)
	enc.AddString("KUBECONFIG_IMAGE_PULL_SECRET_NAME", util.MaskOut(r.KubeconfigImagePullSecretName))
	enc.AddString("KUBECONFIG_PULL_SECRET_NAME", r.KubeconfigPullSecretName)
	enc.AddString("KUBECONFIG_PULL_SECRET_CONTENT", util.MaskOut(r.KubeconfigPullSecretContent))
	return nil
}

type RegisterYugawareRequest struct {
	Code            string `json:"code"`
	Name            string `json:"name"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirmPassword"`
	ConfirmEula     bool   `json:"confirmEula"`
}

type RegisterYugawareResponse AuthToken

type KubernetesRegion struct {
	Code      string           `json:"code"`
	Name      string           `json:"name"`
	Latitude  float32          `json:"latitude"`
	Longitude float32          `json:"longitude"`
	ZoneList  []KubernetesZone `json:"zoneList"`
}

func (r *KubernetesRegion) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("code", r.Code)
	enc.AddString("name", r.Name)
	enc.AddFloat32("latitude", r.Latitude)
	enc.AddFloat32("longitude", r.Longitude)

	return enc.AddArray("zoneList", zapcore.ArrayMarshalerFunc(func(aenc zapcore.ArrayEncoder) error {
		for _, zone := range r.ZoneList {
			err := aenc.AppendObject(&zone)
			if err != nil {
				return err
			}
		}
		return nil
	}))
}

type KubernetesZone struct {
	Code   string               `json:"code"`
	Name   string               `json:"name"`
	Config KubernetesZoneConfig `json:"config"`
}

func (r *KubernetesZone) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("code", r.Code)
	enc.AddString("name", r.Name)
	return enc.AddObject("config", &r.Config)
}

type KubernetesZoneConfig struct {
	StorageClass      string `json:"STORAGE_CLASS"`
	Overrides         string `json:"OVERRIDES"`
	KubeconfigName    string `json:"KUBECONFIG_NAME"`
	KubeconfigContent string `json:"KUBECONFIG_CONTENT"`
}

func (r *KubernetesZoneConfig) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("STORAGE_CLASS", r.StorageClass)
	enc.AddString("OVERRIDES", r.Overrides)
	enc.AddString("KUBECONFIG_NAME", r.KubeconfigName)
	enc.AddString("KUBECONFIG_CONTENT", util.MaskOut(r.KubeconfigContent))

	return nil
}

type ConfigureKubernetesProviderRequest struct {
	Code       string             `json:"code"`
	Name       string             `json:"name"`
	Config     KubernetesConfig   `json:"config"`
	RegionList []KubernetesRegion `json:"regionList"`
}

func (r *ConfigureKubernetesProviderRequest) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("code", r.Code)
	enc.AddString("name", r.Name)

	err := enc.AddObject("config", &r.Config)
	if err != nil {
		return err
	}

	return enc.AddArray("regionList", zapcore.ArrayMarshalerFunc(func(aenc zapcore.ArrayEncoder) error {
		for _, region := range r.RegionList {
			err := aenc.AppendObject(&region)
			if err != nil {
				return err
			}
		}
		return nil
	}))
}

type ConfigureKubernetesProviderResponse Provider
