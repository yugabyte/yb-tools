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
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/universe_management"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/cmdutil"
)

func DeleteUniverseCmd(ctx *cmdutil.YWClientContext) *cobra.Command {
	options := &DeleteOptions{}
	cmd := &cobra.Command{
		Use:   "delete (UNIVERSE_NAME|UNIVERSE_UUID)",
		Short: "Delete a Yugabyte universe",
		Long:  `Delete a Yugabyte universe`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := ctx.WithCmd(cmd).WithOptions(options).Setup()
			if err != nil {
				return err
			}

			// Positional argument
			options.UniverseIdentifier = args[0]

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

func deleteUniverse(ctx *cmdutil.YWClientContext, options *DeleteOptions) error {
	log := ctx.Log

	if !options.Approve {
		err := util.ConfirmationDialog()
		if err != nil {
			return err
		}
	}

	params := options.GetUniverseDeleteParams(ctx)

	log.V(1).Info("deleting universe", "universe_name", options.universe.Name, "universe_uuid", options.universe.UniverseUUID)
	task, err := ctx.Client.PlatformAPIs.UniverseManagement.DeleteUniverse(params, ctx.Client.SwaggerAuth)
	if err != nil {
		return err
	}

	log.V(1).Info("delete universe task", "task", task.GetPayload())

	if options.Wait {
		err = cmdutil.WaitForTaskCompletion(ctx, ctx.Client, task.GetPayload())
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
	UniverseIdentifier string

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

func (o *DeleteOptions) Validate(ctx *cmdutil.YWClientContext) error {
	err := o.validateUniverseIdentifier(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (o *DeleteOptions) Complete(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("a UNIVERSE_NAME or UNIVERSE_UUID argument is required")
	}

	if len(args) > 1 {
		return fmt.Errorf("too many arguments")
	}

	o.UniverseIdentifier = args[0]

	return nil
}

func (o *DeleteOptions) validateUniverseIdentifier(ctx *cmdutil.YWClientContext) error {
	validateUniverseIdentifierError := func(err error) error {
		return fmt.Errorf(`unable to validate universe identifier "%s": %w`, o.UniverseIdentifier, err)
	}

	if o.UniverseIdentifier == "" {
		return validateUniverseIdentifierError(fmt.Errorf(`a UNIVERSE_NAME or UNIVERSE_UUID argument is required`))
	}

	ctx.Log.V(1).Info("fetching universes")

	universe, err := ctx.Client.GetUniverseByIdentifier(o.UniverseIdentifier)
	if err != nil {
		return validateUniverseIdentifierError(err)
	}

	if universe != nil {
		ctx.Log.V(1).Info("got universe", "universe", universe)
		o.universe = universe
		return nil
	}

	return validateUniverseIdentifierError(fmt.Errorf(`universe with identifier "%s" does not exist`, o.UniverseIdentifier))
}

func (o *DeleteOptions) GetUniverseDeleteParams(ctx *cmdutil.YWClientContext) *universe_management.DeleteUniverseParams {
	return universe_management.NewDeleteUniverseParams().
		WithDefaults().
		WithCUUID(ctx.Client.CustomerUUID()).
		WithUniUUID(o.universe.UniverseUUID).
		WithIsForceDelete(&o.Force).
		WithIsDeleteBackups(&o.DeleteBackups)
}
