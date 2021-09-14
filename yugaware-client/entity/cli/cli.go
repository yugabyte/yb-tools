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

package cli

import "go.uber.org/zap/zapcore"

type YugawareGlobalConfig struct {
	Registration        YugawareRegistration `json:"registration"`
	KubernetesProviders []KubernetesProvider `json:"kubernetes_providers"`
}

func (r *YugawareGlobalConfig) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	err := enc.AddObject("registration", &r.Registration)
	if err != nil {
		return err
	}
	return enc.AddReflected("kubernetes_providers", &r.KubernetesProviders)
}

type YugawareRegistration struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password,omitempty"`
}

func (r *YugawareRegistration) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("code", r.Code)

	enc.AddString("name", r.Name)

	enc.AddString("email", r.Email)

	if r.Password != "" {
		enc.AddString("password", "*********")
	}

	return nil
}

type Config struct {
	StorageClass   string `json:"storage_class,omitempty"`
	Overrides      string `json:"overrides,omitempty"`
	KubeconfigPath string `json:"kubeconfig_path,omitempty"`
}

type ZoneInfo struct {
	Name   string `json:"name"`
	Config Config `json:"config,omitempty"`
}

type Regions struct {
	Code     string     `json:"code"`
	ZoneInfo []ZoneInfo `json:"zone_info"`
}

type KubernetesProvider struct {
	Name                string    `json:"name"`
	KubeconfigPath      string    `json:"kubeconfig_path"`
	ServiceAccountName  string    `json:"service_account_name"`
	ImageRegistry       string    `json:"image_registry,omitempty"`
	ImagePullSecretPath string    `json:"image_pull_secret_path,omitempty"`
	Regions             []Regions `json:"regions"`
}
