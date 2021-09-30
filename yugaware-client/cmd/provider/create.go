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

package provider

import (
	"github.com/spf13/cobra"
	"github.com/yugabyte/yb-tools/yugaware-client/cmd/provider/create"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/cmdutil"
)

func CreateCmd(ctx *cmdutil.YWClientContext) *cobra.Command {
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create yugaware providers",
		Long:  `Create yugaware providers`,
	}
	createCmd.Run = createCmd.HelpFunc()

	subcommands := []*cobra.Command{
		create.KubernetesProviderCmd(ctx),
	}

	for _, subcommand := range subcommands {
		createCmd.AddCommand(subcommand)
	}

	return createCmd
}
