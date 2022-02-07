/*
Copyright © 2021-2022 Yugabyte Support

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

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yugabyte/yb-tools/pkg/format"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/session_management"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/cmdutil"
)

func VersionCmd(ctx *cmdutil.YWClientContext) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the client and server version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			err := ctx.WithCmd(cmd).SetupNoClient()
			if err != nil {
				return err
			}

			return version(ctx)
		},
	}
}

func version(ctx *cmdutil.YWClientContext) error {
	type VersionStruct struct {
		ClientVersion string `json:"clientVersion"`
		ServerVersion string `json:"serverVersion,omitempty"`
	}

	serverVersion, err := getServerVersion(ctx)
	if err != nil {
		ctx.Log.Error(err, "could not get server version")
	}

	version := &VersionStruct{
		ClientVersion: Version,
		ServerVersion: serverVersion,
	}

	table := &format.Output{
		JSONObject: version,
		OutputType: ctx.GlobalOptions.Output,
		TableColumns: []format.Column{
			{Name: "CLIENT_VERSION", JSONPath: "$.clientVersion"},
			{Name: "SERVER_VERSION", JSONPath: "$.serverVersion"},
		},
	}
	return table.Print()
}

func getServerVersion(ctx *cmdutil.YWClientContext) (string, error) {
	var version string

	ywc, err := cmdutil.ConnectToYugaware(ctx)
	if err != nil {
		return version, err
	}

	params := session_management.NewAppVersionParams().
		WithDefaults()
	response, err := ywc.PlatformAPIs.SessionManagement.AppVersion(params)
	if err != nil {
		return version, err
	}

	var ok bool
	if version, ok = response.GetPayload()["version"]; !ok {
		return version, fmt.Errorf("app version response did not contain version string")
	}
	return version, nil
}
