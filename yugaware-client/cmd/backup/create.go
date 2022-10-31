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
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yugabyte/yb-tools/pkg/flag"
	"github.com/yugabyte/yb-tools/pkg/format"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/backups"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/cmdutil"
)

func CreateCmd(ctx *cmdutil.YWClientContext) *cobra.Command {
	options := &CreateOptions{}
	cmd := &cobra.Command{
		Use:   "create (UNIVERSE_NAME|UNIVERSE_UUID) --storage-config <config>",
		Short: "Create a backup",
		Long:  `Create a backup`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := ctx.WithCmd(cmd).WithOptions(options).Setup()
			if err != nil {
				return err
			}

			options.UniverseName = args[0]

			err = options.Validate(ctx)
			if err != nil {
				return err
			}

			return create(ctx, options)
		},
	}
	options.AddFlags(cmd)

	return cmd
}

type CreateOptions struct {
	UniverseName string // positional arg

	// Common options
	Type          string        `mapstructure:"type,omitempty"`
	StorageConfig string        `mapstructure:"storage_config,omitempty"`
	Retention     time.Duration `mapstructure:"retention,omitempty"`
	Parallelism   int32         `mapstructure:"parallelism,omitempty"`
	Wait          bool          `mapstructure:"wait,omitempty"`

	// CQL only options
	Keyspace string   `mapstructure:"keyspace,omitempty"`
	Tables   []string `mapstructure:"tables,omitempty"`

	// SQL only options
	Database string `mapstructure:"database,omitempty"`

	universe *models.UniverseResp
	storage  *models.CustomerConfigUI
}

func (o *CreateOptions) AddFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.StringVar(&o.Type, "type", "sql", "Type of backup [sql cql]")
	flags.StringVar(&o.StorageConfig, "storage-config", "", "The storage type to use for the backup")
	flags.DurationVar(&o.Retention, "retention", time.Hour*24*7, "backup retention duration")
	flags.BoolVar(&o.Wait, "wait", false, "Wait for backup completion")
	flags.Int32Var(&o.Parallelism, "parallelism", 3, "The number of threads for backup/restore (per node)")

	flags.StringArrayVar(&o.Tables, "table", []string{}, "Tables to back up (CQL only)")
	flags.StringVar(&o.Keyspace, "keyspace", "", "The keyspace to back up (CQL only)")

	flags.StringVar(&o.Database, "database", "", "The database to back up (SQL only)")

	flag.MarkFlagsRequired([]string{"storage-config"}, flags)
}

func (o *CreateOptions) Validate(ctx *cmdutil.YWClientContext) error {
	o.Type = strings.ToUpper(o.Type)

	err := o.validateUniverseName(ctx)
	if err != nil {
		return err
	}

	if o.Type == "SQL" {
		err := o.validateCreateOptionsSQL()
		if err != nil {
			return err
		}
	} else if o.Type == "CQL" {
		err := o.validateCreateOptionsCQL()
		if err != nil {
			return err
		}
	} else {
		return errors.New(`required flag "type" must be one of [sql cql]`)
	}

	return o.validateStorageOptions(ctx)
}

func (o *CreateOptions) validateUniverseName(ctx *cmdutil.YWClientContext) error {
	validateUniverseNameError := func(err error) error {
		return fmt.Errorf(`unable to validate universe name "%s": %w`, o.UniverseName, err)
	}

	if o.UniverseName == "" {
		return validateUniverseNameError(fmt.Errorf(`required flag "universe-name" is not set`))
	}

	var err error
	o.universe, err = ctx.Client.GetUniverseByIdentifier(o.UniverseName)
	if err != nil {
		return validateUniverseNameError(err)
	}

	if o.universe == nil {
		return validateUniverseNameError(fmt.Errorf("universe does not exist"))
	}

	return nil
}

func (o *CreateOptions) validateCreateOptionsSQL() error {
	if o.Keyspace != "" {
		return errors.New(`the "keyspace" flag is incompatible with an SQL backup`)
	}

	if len(o.Tables) != 0 {
		return errors.New(`table level SQL backups are unsupported`)
	}
	return nil
}

func (o *CreateOptions) validateCreateOptionsCQL() error {
	if o.Database != "" {
		return errors.New(`the "database" flag is incompatible with a CQL backup`)
	}

	if len(o.Tables) != 0 {
		if o.Keyspace == "" {
			return errors.New(`a keyspace must be specified when creating a table level backup`)
		}
	}

	return nil
}

func (o *CreateOptions) validateStorageOptions(ctx *cmdutil.YWClientContext) error {
	var err error
	o.storage, err = ctx.Client.GetStorageConfigByIdentifier(o.StorageConfig)

	if o.storage == nil {
		return fmt.Errorf(`storage config "%s" does not exist`, o.StorageConfig)
	}
	return err
}

func (o *CreateOptions) getCreateMultiTableBackupParams(ctx *cmdutil.YWClientContext) *backups.CreateMultiTableBackupParams {
	var nodeCount int32
	for _, node := range o.universe.UniverseDetails.NodeDetailsSet {
		if node.IsTserver {
			nodeCount++
		}
	}

	parallelism := nodeCount * o.Parallelism

	var keyspace, backupType string
	if o.Type == "SQL" {
		backupType = models.BackupRespBackupTypePGSQLTABLETYPE
		keyspace = o.Database
	} else if o.Type == "CQL" {
		backupType = models.BackupRespBackupTypeYQLTABLETYPE
		keyspace = o.Keyspace
	}

	return backups.NewCreateMultiTableBackupParams().
		WithCUUID(ctx.Client.CustomerUUID()).
		WithUniUUID(o.universe.UniverseUUID).
		WithTableBackup(&models.MultiTableBackupRequestParams{
			ActionType:          models.BackupTableParamsActionTypeCREATE,
			BackupType:          backupType,
			Keyspace:            keyspace,
			Parallelism:         parallelism,
			StorageConfigUUID:   &o.storage.ConfigUUID,
			TransactionalBackup: true,
			TimeBeforeDelete:    int64(o.Retention / time.Millisecond),
		})
}

var _ cmdutil.CommandOptions = &CreateOptions{}

func create(ctx *cmdutil.YWClientContext, options *CreateOptions) error {
	params := options.getCreateMultiTableBackupParams(ctx)

	ctx.Log.V(1).Info("creating backup", "params", params)
	task, err := ctx.Client.PlatformAPIs.Backups.CreateMultiTableBackup(params, ctx.Client.SwaggerAuth)
	if err != nil {
		return err
	}

	if options.Wait {
		err = cmdutil.WaitForTaskCompletion(ctx, ctx.Client, task.GetPayload())
		if err != nil {
			return err
		}
		ctx.Log.V(1).Info("backup complete", "task", task.GetPayload())
	}

	backupInfo, err := waitForBackupSubmitComplete(ctx, options.universe, task.GetPayload())
	if err != nil {
		return err
	}

	table := &format.Output{
		OutputMessage: "Created Backup",
		JSONObject:    backupInfo,
		OutputType:    ctx.GlobalOptions.Output,
		TableColumns: []format.Column{
			{Name: "BACKUP_UUID", JSONPath: "$.backupUUID"},
			{Name: "TYPE", JSONPath: "$.backupInfo.backupType"},
			{Name: "KEYSPACE", JSONPath: "$..keyspace"},
			{Name: "CREATED", Expr: "localtime(@.createTime/1000)"},
			{Name: "EXPIRES", Expr: "localtime(@.expiry/1000)"},
			{Name: "ACTION", JSONPath: "$.backupInfo.actionType"},
			{Name: "STATE", JSONPath: "$.state"},
		},
	}

	return table.Print()
}

func waitForBackupSubmitComplete(ctx *cmdutil.YWClientContext, universe *models.UniverseResp, waitTask *models.YBPTask) ([]*models.Backup, error) {
	timeout := time.After(15 * time.Second)

	for {
		select {
		case <-timeout:
			return nil, errors.New("timed out waiting for backup to start")
		case <-time.After(1 * time.Second):
			fetchBackupParams := backups.NewFetchBackupsByTaskUUIDParams().
				WithCUUID(ctx.Client.CustomerUUID()).
				WithUniUUID(universe.UniverseUUID).
				WithTUUID(waitTask.TaskUUID)

			backupInfo, err := ctx.Client.PlatformAPIs.Backups.FetchBackupsByTaskUUID(fetchBackupParams, ctx.Client.SwaggerAuth)
			if err != nil {
				return nil, err
			}

			if len(backupInfo.GetPayload()) > 0 {
				return backupInfo.GetPayload(), nil
			}

		case <-ctx.Done():
			ctx.Client.Log.Info("wait cancelled")
			return nil, nil
		}
	}
}
