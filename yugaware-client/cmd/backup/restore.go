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

package backup

import (
	"fmt"

	"github.com/go-openapi/strfmt"
	"github.com/spf13/cobra"
	"github.com/yugabyte/yb-tools/pkg/format"
	"github.com/yugabyte/yb-tools/pkg/ybversion"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/session_management"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/table_management"

	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/backups"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/cmdutil"
)

func RestoreCmd(ctx *cmdutil.YWClientContext) *cobra.Command {
	options := &RestoreOptions{}
	cmd := &cobra.Command{
		Use:   "restore BACKUP_UUID",
		Short: "Restore a backup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := ctx.WithCmd(cmd).WithOptions(options).Setup()
			if err != nil {
				return err
			}

			options.BackupUUID = args[0]

			err = options.Validate(ctx)
			if err != nil {
				return err
			}

			return restore(ctx, options)
		},
	}
	options.AddFlags(cmd)

	return cmd
}

type RestoreOptions struct {
	BackupUUID string // positional arg

	UniverseName string `mapstructure:"universe_name"`
	Parallelism  int32  `mapstructure:"parallelism"`
	Wait         bool   `mapstructure:"wait"`

	// TODO: What options should we allow here?
	//Keyspace      string
	//ToKeyspace    string

	backup   *models.Backup
	universe *models.UniverseResp
}

type keyspaceType struct {
	TableType string
	KeySpace  string
}

func (o *RestoreOptions) AddFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	flags.StringVar(&o.UniverseName, "universe", "", "The universe to restore to")
	flags.Int32Var(&o.Parallelism, "parallelism", 3, "The number of threads for backup/restore (per node)")
	flags.BoolVar(&o.Wait, "wait", false, "Wait for the restore to complete")

	// TODO: allow the keyspace to be renamed
	//flags.StringVar(&o.Keyspace, "keyspace", "", "The keyspace restore")
	//flags.StringVar(&o.ToKeyspace, "to-keyspace", "", "The keyspace name to use for the new tables")
}

func (o *RestoreOptions) Validate(ctx *cmdutil.YWClientContext) error {
	err := o.validateVersion(ctx)
	if err != nil {
		return err
	}

	// Step 1: Check to see if the backup exists
	err = o.validateBackup(ctx)
	if err != nil {
		return err
	}

	// Step 2: Check restore destination to see if it exists
	err = o.validateUniverse(ctx)
	if err != nil {
		return err
	}

	// Step 3: Validate whether that the keyspace exists at the destination, and refuse to restore if the keyspace exists.
	return o.validateKeyspaces(ctx)

	// TODO: validate KMS configs
}

func (o *RestoreOptions) validateVersion(ctx *cmdutil.YWClientContext) error {
	params := session_management.NewAppVersionParams().WithDefaults()
	response, err := ctx.Client.PlatformAPIs.SessionManagement.AppVersion(params)
	if err != nil {
		return err
	}
	if v, ok := response.GetPayload()["version"]; ok {
		version, err := ybversion.New(v)
		if err != nil {
			return err
		}
		// Backup V2 interfaces were first added in v2.13.0
		if version.Lt(ybversion.YBVersion{Major: 2, Minor: 13}) {
			return fmt.Errorf(`server version "%s": restore is only supported in version 2.13.0 and above`, v)
		}
		return nil
	}
	return fmt.Errorf("app version response did not contain version string")
}
func (o *RestoreOptions) validateBackup(ctx *cmdutil.YWClientContext) error {
	params := backups.NewGetBackupV2Params().
		WithCUUID(ctx.Client.CustomerUUID()).
		WithBackupUUID(strfmt.UUID(o.BackupUUID))

	backupResponse, err := ctx.Client.PlatformAPIs.Backups.GetBackupV2(params, ctx.Client.SwaggerAuth)
	if err != nil {
		return err
	}
	o.backup = backupResponse.GetPayload()
	return nil
}

func (o *RestoreOptions) validateUniverse(ctx *cmdutil.YWClientContext) error {
	var err error

	universe := string(o.backup.UniverseUUID)
	if o.UniverseName != "" {
		universe = o.UniverseName
	}

	o.universe, err = ctx.Client.GetUniverseByIdentifier(universe)
	if err != nil {
		return err
	}

	if o.universe == nil {
		return fmt.Errorf("universe %s does not exist", universe)
	}

	return nil
}

func (o *RestoreOptions) validateKeyspaces(ctx *cmdutil.YWClientContext) error {
	validateKeyspacesError := func(err error) error {
		return fmt.Errorf(`cannot restore backup "%s" to universe "%s": %w`, o.BackupUUID, o.universe.Name, err)
	}
	params := table_management.NewGetAllTablesParams().
		WithCUUID(ctx.Client.CustomerUUID()).
		WithUniUUID(o.universe.UniverseUUID)

	tablesResponse, err := ctx.Client.PlatformAPIs.TableManagement.GetAllTables(params, ctx.Client.SwaggerAuth)
	if err != nil {
		return validateKeyspacesError(err)
	}

	keyspacesInTargetUniverse := map[keyspaceType]bool{}
	for _, table := range tablesResponse.GetPayload() {
		keyspacesInTargetUniverse[keyspaceType{
			KeySpace:  table.KeySpace,
			TableType: table.TableType,
		}] = true
	}

	storage := getBackupStorageInfo(o.backup)

	for _, backupInfo := range storage {
		if _, ok := keyspacesInTargetUniverse[keyspaceType{
			TableType: backupInfo.BackupType,
			KeySpace:  backupInfo.Keyspace,
		}]; ok {
			keyspaceLabel := "keyspace"
			if o.backup.BackupInfo.BackupType == models.BackupTableParamsBackupTypePGSQLTABLETYPE {
				keyspaceLabel = "database"
			}

			// TODO: allow the keyspace to be renamed
			return validateKeyspacesError(fmt.Errorf(`%s "%s" already exists`, keyspaceLabel, backupInfo.Keyspace))
		}
	}

	return nil
}

func (o *RestoreOptions) getRestoreParams(ctx *cmdutil.YWClientContext) *backups.RestoreBackupV2Params {
	var nodeCount int32
	for _, node := range o.universe.UniverseDetails.NodeDetailsSet {
		if node.IsTserver {
			nodeCount++
		}
	}

	parallelism := nodeCount * o.Parallelism

	return backups.NewRestoreBackupV2Params().
		WithCUUID(ctx.Client.CustomerUUID()).
		WithBackup(&models.RestoreBackupParams{
			ActionType:            models.BackupTableParamsActionTypeRESTORE,
			BackupStorageInfoList: getBackupStorageInfo(o.backup),
			CustomerUUID:          ctx.Client.CustomerUUID(),
			StorageConfigUUID:     *o.backup.BackupInfo.StorageConfigUUID,
			UniverseUUID:          &o.universe.UniverseUUID,
			Parallelism:           parallelism,
		})
}
func getBackupStorageInfo(backup *models.Backup) []*models.BackupStorageInfo {
	var storageInfo []*models.BackupStorageInfo

	if backup.BackupInfo.StorageLocation != "" {
		storageInfo = append(storageInfo, &models.BackupStorageInfo{
			BackupType:      backup.BackupInfo.BackupType,
			Keyspace:        backup.BackupInfo.Keyspace,
			Sse:             backup.BackupInfo.Sse,
			StorageLocation: backup.BackupInfo.StorageLocation,
			TableNameList:   backup.BackupInfo.TableNameList, // TODO: filter by table name in CQL type backups?
		})
	}

	for _, backupList := range backup.BackupInfo.BackupList {
		storageInfo = append(storageInfo, &models.BackupStorageInfo{
			BackupType:      backupList.BackupType,
			Keyspace:        backupList.Keyspace,
			Sse:             backupList.Sse,
			StorageLocation: backupList.StorageLocation,
			TableNameList:   backupList.TableNameList,
		})
	}

	return storageInfo
}

var _ cmdutil.CommandOptions = &RestoreOptions{}

func restore(ctx *cmdutil.YWClientContext, options *RestoreOptions) error {
	restoreParams := options.getRestoreParams(ctx)

	task, err := ctx.Client.PlatformAPIs.Backups.RestoreBackupV2(restoreParams, ctx.Client.SwaggerAuth)
	if err != nil {
		return err
	}

	action := "Started"
	if options.Wait {
		err = cmdutil.WaitForTaskCompletion(ctx, ctx.Client, task.GetPayload())
		if err != nil {
			return err
		}
		action = "Completed"
	}

	table := &format.Output{
		OutputMessage: fmt.Sprintf("Restore %s", action),
		JSONObject:    task.GetPayload(),
		OutputType:    ctx.GlobalOptions.Output,
		TableColumns: []format.Column{
			{Name: "TASK_UUID", JSONPath: "$.taskUUID"},
		},
	}
	return table.Print()
}
