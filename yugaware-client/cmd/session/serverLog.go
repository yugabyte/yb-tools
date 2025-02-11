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

package session

import (
	"fmt"

	"github.com/yugabyte/yb-tools/pkg/format"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/session_management"

	"github.com/spf13/cobra"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/cmdutil"
)

func ServerLogCmd(ctx *cmdutil.YWClientContext) *cobra.Command {
	options := &ServerLogOptions{}

	cmd := &cobra.Command{
		Use:   "server_log",
		Short: "Display the the most recent --lines=<number> from the YW server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			err := ctx.WithCmd(cmd).WithOptions(options).Setup()
			if err != nil {
				return err
			}

			return serverLog(ctx, options.Lines)
		},
	}
	options.AddFlags(cmd)

	return cmd
}

var _ cmdutil.CommandOptions = &ServerLogOptions{}

type ServerLogOptions struct {
	Lines uint `mapstructure:"lines,omitempty"`
}

func (o *ServerLogOptions) Validate(_ *cmdutil.YWClientContext) error {
	return nil
}

func (o *ServerLogOptions) AddFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	flags.UintVar(&o.Lines, "lines", 50, "Number of log lines to display")
}

func serverLog(ctx *cmdutil.YWClientContext, logLines uint) error {
	params := session_management.NewGetLogsParams().
		WithMaxLines(int32(logLines))

	response, err := ctx.Client.PlatformAPIs.SessionManagement.GetLogs(params, ctx.Client.SwaggerAuth)
	if err != nil {
		return err
	}
	if response == nil {
		return fmt.Errorf("unable to retrieve server logs: %w", err)
	}

	table := &format.Output{
		JSONObject: response.GetPayload(),
		OutputType: ctx.GlobalOptions.Output,
	}
	if ctx.GlobalOptions.Output != "table" {
		return table.Println()
	}
	payload := response.GetPayload()

	if payload != nil {
		for _, line := range payload.Lines {
			ctx.Cmd.Println(line)
		}
	}

	return nil
}
