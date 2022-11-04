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

package backup

import (
	"fmt"

	"github.com/go-openapi/strfmt"
	"github.com/spf13/cobra"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/universe_management"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"

	"github.com/yugabyte/yb-tools/pkg/format"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/backups"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/cmdutil"
)

func ListCmd(ctx *cmdutil.YWClientContext) *cobra.Command {
	options := &ListOptions{}

	cmd := &cobra.Command{
		Use:   "list [UNIVERSE_NAME|UNIVERSE_IDENTIFIER]...",
		Short: "List backups for a Yugabyte universe",
		Long:  `List backups for a Yugabyte universe`,
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, universes []string) error {
			err := ctx.WithCmd(cmd).WithOptions(options).Setup()
			if err != nil {
				return err
			}

			return list(ctx, universes, options)
		},
	}
	options.AddFlags(cmd)

	return cmd
}

type ListOptions struct {
	IncludeRestore bool `mapstructure:"include_restore,omitempty"`
}

var _ cmdutil.CommandOptions = &ListOptions{}

func (o *ListOptions) AddFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	// Create flags
	flags.BoolVar(&o.IncludeRestore, "include-restore", false, "include restores in the backup list")
}

func (o *ListOptions) Validate(_ *cmdutil.YWClientContext) error {
	return nil
}

func matchesUniverse(resp *models.UniverseResp, universeIdentifier string) bool {
	if resp.Name == universeIdentifier {
		return true
	}

	if resp.UniverseUUID == strfmt.UUID(universeIdentifier) {
		return true
	}

	return false
}

func shouldListBackups(universe *models.UniverseResp, listOfUniverses []string) bool {
	hasFilter := len(listOfUniverses) > 0

	if hasFilter {
		for _, uid := range listOfUniverses {
			if matchesUniverse(universe, uid) {
				return true
			}
		}
		return false
	}
	return true
}

func list(ctx *cmdutil.YWClientContext, universes []string, options *ListOptions) error {
	listParams := universe_management.NewListUniversesParams().
		WithCUUID(ctx.Client.CustomerUUID())

	universesResp, err := ctx.Client.PlatformAPIs.UniverseManagement.ListUniverses(listParams, ctx.Client.SwaggerAuth)
	if err != nil {
		return err
	}

	for _, universe := range universesResp.GetPayload() {
		if !shouldListBackups(universe, universes) {
			continue
		}

		ctx.Log.V(1).Info("fetching backups", "universe", universe)
		params := backups.NewListOfBackupsParams().
			WithCUUID(ctx.Client.CustomerUUID()).
			WithUniUUID(universe.UniverseUUID)

		backups, err := ctx.Client.PlatformAPIs.Backups.ListOfBackups(params, ctx.Client.SwaggerAuth)
		if err != nil {
			return err
		}

		table := &format.Output{
			OutputMessage: fmt.Sprintf(`Backup list for universe "%s"`, universe.Name),
			JSONObject:    backups.GetPayload(),
			OutputType:    ctx.GlobalOptions.Output,
			TableColumns: []format.Column{
				{Name: "BACKUP_UUID", JSONPath: "$.backupUUID"},
				{Name: "TYPE", JSONPath: "$.backupInfo.backupType"},
				{Name: "KEYSPACE", JSONPath: "$..keyspace"},
				{Name: "CREATED", Expr: "localtime(@.createTime/1000)"},
				{Name: "EXPIRES", Expr: "localtime(@.expiry/1000)"},
				{Name: "SIZE", Expr: "size_pretty(@.backupInfo.backupSizeInBytes)"},
				{Name: "ACTION", JSONPath: "$.backupInfo.actionType"},
				{Name: "STATE", JSONPath: "$.state"},
			},
		}

		if !options.IncludeRestore {
			table.Filter = "@.backupInfo.actionType != 'RESTORE'"
		}

		err = table.Println()
		if err != nil {
			ctx.Log.Error(err, "could not list backups for universe", "universe", universe)
		}
	}

	return nil
}
