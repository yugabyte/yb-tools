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

package certificate

import (
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/certificate_info"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/cmdutil"
)

func GetRootCmd(ctx *cmdutil.YWClientContext) *cobra.Command {
	return &cobra.Command{
		Use:   "get-root <CERT_UUID> | <CERT_NAME>",
		Short: "List client certificates",
		Long:  `List client certificates`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := ctx.WithCmd(cmd).Setup()
			if err != nil {
				return err
			}

			certIdentifier := args[0]

			cert, err := ctx.Client.GetCertByIdentifier(certIdentifier)
			if err != nil {
				return err
			}

			if cert == nil {
				return fmt.Errorf("certificate %s does not exist", certIdentifier)
			}

			return get(ctx, cert)
		},
	}
}

func get(ctx *cmdutil.YWClientContext, certificateInfo *models.CertificateInfo) error {
	log := ctx.Log
	ywc := ctx.Client

	log.V(1).Info("fetching root certificate")

	params := certificate_info.NewGetRootCertParams().
		WithCUUID(ywc.CustomerUUID()).
		WithRUUID(certificateInfo.UUID)

	certificate, err := ywc.PlatformAPIs.CertificateInfo.GetRootCert(params, ywc.SwaggerAuth)
	if err != nil {
		return err
	}

	certJSON, err := json.MarshalIndent(certificate.GetPayload(), " ", " ")
	if err != nil {
		return err
	}

	if ctx.GlobalOptions.Output == "json" {
		fmt.Println(string(certJSON))
	} else if ctx.GlobalOptions.Output == "yaml" {
		certYAML, err := yaml.JSONToYAML(certJSON)
		if err != nil {
			return err
		}
		fmt.Print(string(certYAML))
	} else {
		fmt.Print(certificate.GetPayload().RootCrt)
	}

	return nil
}
