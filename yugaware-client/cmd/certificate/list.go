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
	"github.com/spf13/cobra"
	"github.com/yugabyte/yb-tools/pkg/format"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/certificate_info"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/cmdutil"
)

func ListCmd(ctx *cmdutil.YWClientContext) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List client certificates",
		Long:  `List client certificates`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			err := ctx.WithCmd(cmd).Setup()
			if err != nil {
				return err
			}

			return list(ctx)
		},
	}
}

func list(ctx *cmdutil.YWClientContext) error {
	log := ctx.Log
	ywc := ctx.Client

	log.V(1).Info("fetching certificates")
	params := certificate_info.NewGetListOfCertificateParams().WithCUUID(ywc.CustomerUUID())

	certificates, err := ywc.PlatformAPIs.CertificateInfo.GetListOfCertificate(params, ywc.SwaggerAuth)
	if err != nil {
		return err
	}

	table := &format.Output{
		OutputMessage: "Certificate List",
		JSONObject:    certificates.GetPayload(),
		OutputType:    ctx.GlobalOptions.Output,
		TableColumns: []format.Column{
			{Name: "LABEL", JSONPath: "$.label"},
			{Name: "TYPE", JSONPath: "$.certType"},
			{Name: "UUID", JSONPath: "$.uuid"},
			{Name: "CREATED", JSONPath: "$.startDate"},
			{Name: "EXPIRES", JSONPath: "$.expiryDate"},
			{Name: "UNIVERSES", JSONPath: "$.universeDetails[*].name"},
		},
	}
	return table.Print()
}
