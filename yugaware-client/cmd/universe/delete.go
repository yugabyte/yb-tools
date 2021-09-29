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

package universe

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yugabyte/yb-tools/pkg/format"
	"github.com/yugabyte/yb-tools/pkg/util"
	cmdutil "github.com/yugabyte/yb-tools/yugaware-client/cmd/util"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/universe_management"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
)

func DeleteUniverseCmd(ctx *cmdutil.CommandContext) *cobra.Command {
	options := &DeleteOptions{}
	cmd := &cobra.Command{
		Use:   "delete UNIVERSE_NAME",
		Short: "Delete a Yugabyte universe",
		Long:  `Delete a Yugabyte universe`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := ctx.WithCmd(cmd).WithOptions(options).Setup()
			if err != nil {
				return err
			}

			// Positional argument
			options.UniverseName = args[0]

			err = options.Validate(ctx)
			if err != nil {
				return err
			}

			return deleteUniverse(ctx, options)
		},
	}

	options.AddFlags(cmd)

	return cmd
}

func deleteUniverse(ctx *cmdutil.CommandContext, options *DeleteOptions) error {
	log := ctx.Log

	if !options.Approve {
		err := util.ConfirmationDialog()
		if err != nil {
			return err
		}
	}

	params := options.GetUniverseDeleteParams(ctx)

	log.V(1).Info("deleting universe", "universe_name", options.UniverseName, "uuid", options.universe.UniverseUUID)
	task, err := ctx.Client.PlatformAPIs.UniverseManagement.DeleteUniverse(params, ctx.Client.SwaggerAuth)
	if err != nil {
		return err
	}

	log.V(1).Info("delete universe task", "task", task.GetPayload())

	if options.Wait {
		err = cmdutil.WaitForTaskCompletion(ctx, task.GetPayload())
		if err != nil {
			return err
		}
	}

	table := &format.Output{
		OutputMessage: "Universe Deleted",
		JSONObject:    task.GetPayload(),
		OutputType:    ctx.GlobalOptions.Output,
		TableColumns: []format.Column{
			{Name: "UNIVERSE_UUID", JSONPath: "$.resourceUUID"},
			{Name: "TASK_UUID", JSONPath: "$.taskUUID"},
		},
	}
	return table.Print()
}

type DeleteOptions struct {
	UniverseName string

	Approve       bool `mapstructure:"approve,omitempty"`
	Force         bool `mapstructure:"force,omitempty"`
	DeleteBackups bool `mapstructure:"delete_backups,omitempty"`
	Wait          bool `mapstructure:"wait,omitempty"`

	universe *models.UniverseResp
}

var _ cmdutil.CommandOptions = &DeleteOptions{}

func (o *DeleteOptions) AddFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	flags.BoolVar(&o.Approve, "approve", false, "Approve delete without prompting")
	flags.BoolVar(&o.Force, "force", false, "Ignore errors and force universe deletion")
	flags.BoolVar(&o.DeleteBackups, "delete-backups", false, "Delete all universe backups")
	flags.BoolVar(&o.Wait, "wait", false, "Wait for create to complete")
}

func (o *DeleteOptions) Validate(ctx *cmdutil.CommandContext) error {
	err := o.validateUniverseName(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (o *DeleteOptions) Complete(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("required argument UNIVERSE_NAME is not set")
	}

	if len(args) > 1 {
		return fmt.Errorf("too many arguments")
	}

	o.UniverseName = args[0]

	return nil
}

func (o *DeleteOptions) validateUniverseName(ctx *cmdutil.CommandContext) error {
	validateUniverseNameError := func(err error) error {
		return fmt.Errorf(`unable to validate universe name "%s": %w`, o.UniverseName, err)
	}

	if o.UniverseName == "" {
		return validateUniverseNameError(fmt.Errorf(`required argument UNIVERSE_NAME is not set`))
	}

	ctx.Log.V(1).Info("fetching universes")
	params := universe_management.NewListUniversesParams().
		WithContext(ctx).
		WithCUUID(ctx.Client.CustomerUUID())
	universes, err := ctx.Client.PlatformAPIs.UniverseManagement.ListUniverses(params, ctx.Client.SwaggerAuth)
	if err != nil {
		return validateUniverseNameError(err)
	}
	ctx.Log.V(1).Info("got universes", "universes", universes.GetPayload())

	for _, universe := range universes.GetPayload() {
		if universe.Name == o.UniverseName {
			o.universe = universe
			return nil
		}
	}
	return validateUniverseNameError(fmt.Errorf(`universe with name "%s" does not exist`, o.UniverseName))
}

func (o *DeleteOptions) GetUniverseDeleteParams(ctx *cmdutil.CommandContext) *universe_management.DeleteUniverseParams {
	return universe_management.NewDeleteUniverseParams().
		WithDefaults().
		WithCUUID(ctx.Client.CustomerUUID()).
		WithUniUUID(o.universe.UniverseUUID).
		WithIsForceDelete(&o.Force).
		WithIsDeleteBackups(&o.DeleteBackups)
}
