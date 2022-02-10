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
	"encoding/json"
	"fmt"

	"github.com/go-openapi/strfmt"
	"github.com/icza/gox/gox"
	"github.com/spf13/cobra"
	"github.com/yugabyte/yb-tools/pkg/format"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/access_keys"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/cloud_providers"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/instance_types"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/region_management"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/release_management"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/session_management"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/universe_cluster_mutations"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/cmdutil"
)

func CreateUniverseCmd(ctx *cmdutil.YWClientContext) *cobra.Command {
	options := &CreateOptions{}
	cmd := &cobra.Command{
		Use:   "create UNIVERSE_NAME --provider <provider> --regions <region> --instance-type <size>",
		Short: "Create a Yugabyte universe",
		Long:  `Create a Yugabyte universe`,
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

			return createUniverse(ctx, options)
		},
	}

	options.AddFlags(cmd)

	return cmd
}

func createUniverse(ctx *cmdutil.YWClientContext, options *CreateOptions) error {
	log := ctx.Log
	ywc := ctx.Client

	params := options.GetUniverseConfigParams(ctx)

	log.V(1).Info("creating universe", "config", params.UniverseConfigureTaskParams.Clusters)
	task, err := ywc.PlatformAPIs.UniverseClusterMutations.CreateAllClusters(params, ywc.SwaggerAuth)
	if err != nil {
		return err
	}

	log.V(1).Info("create universe task", "task", task.GetPayload())

	if options.Wait {
		err = cmdutil.WaitForTaskCompletion(ctx, ctx.Client, task.GetPayload())
		if err != nil {
			return err
		}
	}

	table := &format.Output{
		OutputMessage: "Universe Created",
		JSONObject:    task.GetPayload(),
		OutputType:    ctx.GlobalOptions.Output,
		TableColumns: []format.Column{
			{Name: "UNIVERSE_UUID", JSONPath: "$.resourceUUID"},
			{Name: "TASK_UUID", JSONPath: "$.taskUUID"},
		},
	}
	return table.Print()
}

type CreateOptions struct {
	UniverseName string // positional arg

	Provider          string            `mapstructure:"provider,omitempty"`
	Regions           []string          `mapstructure:"regions,omitempty"`
	PreferredRegion   string            `mapstructure:"preferred_region,omitempty"`
	Version           string            `mapstructure:"version,omitempty"`
	NodeCount         int32             `mapstructure:"node_count,omitempty"`
	ReplicationFactor int32             `mapstructure:"replication_factor,omitempty"`
	DisableYSQL       bool              `mapstructure:"disable_ysql,omitempty"`
	EnableEncryption  bool              `mapstructure:"enable_encryption,omitempty"`
	InstanceType      string            `mapstructure:"instance_type,omitempty"`
	TserverGFlags     map[string]string `mapstructure:"tserver_gflags,omitempty"`
	MasterGFlags      map[string]string `mapstructure:"master_gflags,omitempty"`
	AssignPublicIP    bool              `mapstructure:"assign_public_ip,omitempty"`
	StaticPublicIP    bool              `mapstructure:"static_public_ip,omitempty"`
	UseSystemd        bool              `mapstructure:"use_systemd,omitempty"`
	VolumeSize        int32             `mapstructure:"volume_size,omitempty"`

	Wait bool `mapstructure:"wait,omitempty"`

	provider        *models.Provider
	preferredRegion strfmt.UUID
	regions         []*models.Region
	instanceType    *models.InstanceType
	storageType     string
	storageClass    string
	accessKey       string
}

var _ cmdutil.CommandOptions = &CreateOptions{}

func (o *CreateOptions) AddFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	// Create flags
	flags.StringVar(&o.Provider, "provider", "", "the Yugaware provider to be used for deployment")
	flags.StringArrayVar(&o.Regions, "regions", []string{}, "list of regions to deploy yugabyte")
	flags.StringVar(&o.PreferredRegion, "preferred-region", "", "preferred region for tablet leaders")
	flags.StringVar(&o.Version, "version", "", "the version of Yugabyte to deploy (defaults to yugaware version)")
	flags.Int32Var(&o.NodeCount, "node-count", 3, "number of nodes to deploy")
	flags.Int32Var(&o.ReplicationFactor, "replication-factor", 3, "replication factor for the cluster")
	flags.BoolVar(&o.DisableYSQL, "disable-ysql", false, "disable ysql")
	flags.BoolVar(&o.EnableEncryption, "enable-encryption", false, "enable node-to-node and client-to-node encryption on the cluster")
	flags.StringVar(&o.InstanceType, "instance-type", "", "instance type to use for cluster nodes")
	flags.StringToStringVar(&o.TserverGFlags, "tserver-gflags", map[string]string{}, "key value pairs of tserver gflags")
	flags.StringToStringVar(&o.MasterGFlags, "master-gflags", map[string]string{}, "key value pairs of master gflags")
	flags.BoolVar(&o.AssignPublicIP, "assign-public-ip", false, "assign a public IP address to the cluster")
	flags.BoolVar(&o.StaticPublicIP, "static-public-ip", false, "assign a static public IP to the cluster")
	flags.BoolVar(&o.UseSystemd, "use-systemd", false, "use systemd as the daemon controller")
	flags.Int32Var(&o.VolumeSize, "volume-size", 0, "volume size to use for cluster nodes")

	// Other flags
	flags.BoolVar(&o.Wait, "wait", false, "Wait for create to complete")
}

func (o *CreateOptions) GetUniverseConfigParams(ctx *cmdutil.YWClientContext) *universe_cluster_mutations.CreateAllClustersParams {
	cluster := &models.Cluster{
		ClusterType: gox.NewString(models.ClusterClusterTypePRIMARY),
		UserIntent: &models.UserIntent{
			AccessKeyCode:        o.accessKey,
			AssignPublicIP:       o.AssignPublicIP,
			AssignStaticPublicIP: o.StaticPublicIP,
			AwsArnString:         "",
			DeviceInfo: &models.DeviceInfo{
				NumVolumes:   1,
				StorageClass: o.storageClass,
				StorageType:  o.storageType,
				VolumeSize:   o.VolumeSize,
			},
			EnableClientToNodeEncrypt: o.EnableEncryption,
			EnableExposingService:     models.UserIntentEnableExposingServiceUNEXPOSED, // TODO: Should this be a flag?
			EnableIPV6:                false,
			EnableNodeToNodeEncrypt:   o.EnableEncryption,
			EnableVolumeEncryption:    false,
			EnableYEDIS:               false,
			EnableYSQL:                !o.DisableYSQL,
			InstanceTags:              nil,
			InstanceType:              *o.instanceType.InstanceTypeCode,
			MasterGFlags:              o.MasterGFlags,
			NumNodes:                  o.NodeCount,
			PreferredRegion:           o.preferredRegion,
			Provider:                  o.provider.UUID.String(),
			ProviderType:              o.provider.Code,
			RegionList:                o.getRegionUUIDs(),
			ReplicationFactor:         o.ReplicationFactor,
			TserverGFlags:             o.TserverGFlags,
			UniverseName:              o.UniverseName,
			UseHostname:               false,
			UseSystemd:                o.UseSystemd,
			UseTimeSync:               true, // TODO: should this ever be false?
			YbSoftwareVersion:         o.Version,
		},
	}

	ctx.Log.V(1).Info("generated create cluster request", "cluster", cluster)
	return universe_cluster_mutations.NewCreateAllClustersParams().
		WithContext(ctx).
		WithCUUID(ctx.Client.CustomerUUID()).
		WithUniverseConfigureTaskParams(&models.UniverseConfigureTaskParams{Clusters: []*models.Cluster{cluster}})
}

func (o *CreateOptions) Validate(ctx *cmdutil.YWClientContext) error {
	err := o.validateUniverseName(ctx)
	if err != nil {
		return err
	}

	err = o.validateProvider(ctx)
	if err != nil {
		return err
	}

	err = o.validateVersion(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (o *CreateOptions) validateUniverseName(ctx *cmdutil.YWClientContext) error {
	validateUniverseNameError := func(err error) error {
		return fmt.Errorf(`unable to validate universe name "%s": %w`, o.UniverseName, err)
	}

	if o.UniverseName == "" {
		return validateUniverseNameError(fmt.Errorf(`required flag "universe-name" is not set`))
	}

	ctx.Log.V(1).Info("fetching universes")
	universe, err := ctx.Client.GetUniverseByIdentifier(o.UniverseName)
	if err != nil {
		return validateUniverseNameError(err)
	}

	if universe != nil {
		return validateUniverseNameError(fmt.Errorf(`universe with name "%s" already exists`, universe.Name))
	}
	return nil
}

func (o *CreateOptions) validateProvider(ctx *cmdutil.YWClientContext) error {
	validateProviderError := func(providers []*models.Provider, err error) error {
		var expectedProviderNames []string
		if providers != nil {
			for _, provider := range providers {
				expectedProviderNames = append(expectedProviderNames, provider.Name)
			}

			err = fmt.Errorf("%w - must be one of: %v", err, expectedProviderNames)
		}

		return fmt.Errorf(`unable to validate provider "%s": %w`, o.Provider, err)
	}

	ctx.Log.V(1).Info("fetching providers")
	params := cloud_providers.NewGetListOfProvidersParams().
		WithContext(ctx).
		WithCUUID(ctx.Client.CustomerUUID())
	providers, err := ctx.Client.PlatformAPIs.CloudProviders.GetListOfProviders(params, ctx.Client.SwaggerAuth)
	if err != nil {
		return validateProviderError(nil, err)
	}
	ctx.Log.V(1).Info("got providers", "providers", providers.GetPayload())

	if o.Provider == "" {
		return validateProviderError(providers.GetPayload(), fmt.Errorf(`required flag "provider" is not set`))
	}

	for _, provider := range providers.GetPayload() {
		if provider.Name == o.Provider {
			o.provider = provider

			err = o.validateRegions(ctx)
			if err != nil {
				return err
			}

			err = o.validateInstanceType(ctx)
			if err != nil {
				return err
			}

			err = o.validateStorageClass(ctx)
			if err != nil {
				return err
			}

			err = o.validateVolumeSize(ctx)
			if err != nil {
				return err
			}

			return o.getAccessKey(ctx)
		}
	}

	return validateProviderError(providers.GetPayload(), fmt.Errorf("could not find provider"))
}

func (o *CreateOptions) validateRegions(ctx *cmdutil.YWClientContext) error {
	validateRegionError := func(regions []*models.Region, err error) error {
		var expectedRegionNames []string
		if regions != nil {
			for _, region := range regions {
				expectedRegionNames = append(expectedRegionNames, region.Code)
			}

			err = fmt.Errorf("%w regions must consist of: %v", err, expectedRegionNames)
		}

		return fmt.Errorf("unable to validate regions %v: %w", o.Regions, err)
	}

	ctx.Log.V(1).Info("fetching regions")
	params := region_management.NewGetRegionParams().
		WithContext(ctx).
		WithCUUID(ctx.Client.CustomerUUID()).
		WithPUUID(o.provider.UUID)
	regions, err := ctx.Client.PlatformAPIs.RegionManagement.GetRegion(params, ctx.Client.SwaggerAuth)
	if err != nil {
		return validateRegionError(nil, err)
	}

	if len(o.Regions) == 0 {
		return validateRegionError(regions.GetPayload(), fmt.Errorf(`required flag "regions" not set`))
	}

	ctx.Log.V(1).Info("got regions", "regions", regions.GetPayload())
	for _, region := range regions.GetPayload() {
		for _, expectedRegion := range o.Regions {
			if region.Code == expectedRegion {
				ctx.Log.V(1).Info("found region", "region", region)
				o.regions = append(o.regions, region)

				if o.PreferredRegion == region.Name {
					ctx.Log.V(1).Info("found preferred region", "region", region)
					o.preferredRegion = region.UUID
				}
				continue
			}
		}
	}

	if len(o.Regions) != len(o.regions) {
		return validateRegionError(regions.GetPayload(), fmt.Errorf("region(s) not found"))
	}

	if o.PreferredRegion != "" &&
		o.preferredRegion == "" {
		return validateRegionError(regions.GetPayload(), fmt.Errorf("preferred region %s does not exist", o.PreferredRegion))
	}

	return nil
}

func (o *CreateOptions) validateInstanceType(ctx *cmdutil.YWClientContext) error {
	validateInstanceTypeError := func(instances *[]*models.InstanceType, err error) error {
		var expectedInstanceTypes []string
		if instances != nil {
			for _, region := range *instances {
				expectedInstanceTypes = append(expectedInstanceTypes, *region.InstanceTypeCode)
			}

			err = fmt.Errorf("%w must be one of: %v", err, expectedInstanceTypes)
		}

		return fmt.Errorf(`unable to validate instance type "%s": %w`, o.InstanceType, err)
	}

	ctx.Log.V(1).Info("fetching instance types")
	params := instance_types.NewListOfInstanceTypeParams().
		WithContext(ctx).
		WithCUUID(ctx.Client.CustomerUUID()).
		WithPUUID(o.provider.UUID)
	instanceTypesDetails, err := ctx.Client.PlatformAPIs.InstanceTypes.ListOfInstanceType(params, ctx.Client.SwaggerAuth)
	if err != nil {
		return validateInstanceTypeError(nil, err)
	}
	ctx.Log.V(1).Info("got instance types", "instance_types", instanceTypesDetails.GetPayload())

	instanceTypesJSON, err := json.Marshal(instanceTypesDetails.GetPayload())
	if err != nil {
		return validateInstanceTypeError(nil, err)
	}

	instanceTypes := &[]*models.InstanceType{}
	err = json.Unmarshal(instanceTypesJSON, instanceTypes)
	if err != nil {
		return validateInstanceTypeError(nil, err)
	}

	if o.InstanceType == "" {
		return validateInstanceTypeError(instanceTypes, fmt.Errorf(`required flag "instance-type" not set`))
	}

	for _, instanceType := range *instanceTypes {
		if *instanceType.InstanceTypeCode == o.InstanceType {
			o.instanceType = instanceType
			return nil
		}
	}

	return validateInstanceTypeError(instanceTypes, fmt.Errorf("could not find instance type"))
}

func (o *CreateOptions) validateStorageClass(_ *cmdutil.YWClientContext) error {
	// TODO: Should this be a flag?
	if o.instanceType != nil {
		cloud := o.instanceType.Provider.Code
		if cloud == "gcp" {
			o.storageType = "Persistent"
		} else if cloud == "kubernetes" {
			o.storageClass = "standard"
		}
	}
	return nil
}

func (o *CreateOptions) validateVolumeSize(_ *cmdutil.YWClientContext) error {
	if o.instanceType != nil && o.VolumeSize == 0 {
		if len(o.instanceType.InstanceTypeDetails.VolumeDetailsList) > 0 {
			o.VolumeSize = *o.instanceType.InstanceTypeDetails.VolumeDetailsList[0].VolumeSizeGB
		}
	}

	if o.VolumeSize <= 0 {
		return fmt.Errorf(`must set flag "volume-size"`)
	}
	return nil
}

func (o *CreateOptions) validateVersion(ctx *cmdutil.YWClientContext) error {
	validateReleaseError := func(releases map[string]interface{}, err error) error {
		var expectedReleaseNames []string
		if releases != nil {
			for release := range releases {
				expectedReleaseNames = append(expectedReleaseNames, release)
			}

			err = fmt.Errorf("%w - must be one of: %v", err, expectedReleaseNames)
		}

		return fmt.Errorf(`unable to validate version "%s": %w`, o.Version, err)
	}

	ctx.Log.V(1).Info("fetching releases")
	params := release_management.NewGetListOfReleasesParams().
		WithContext(ctx).
		WithCUUID(ctx.Client.CustomerUUID()).
		WithIncludeMetadata(gox.NewBool(true))
	releases, err := ctx.Client.PlatformAPIs.ReleaseManagement.GetListOfReleases(params, ctx.Client.SwaggerAuth)
	if err != nil {
		return validateReleaseError(nil, err)
	}
	ctx.Log.V(1).Info("got releases", "releases", releases.GetPayload())

	if o.Version == "" {
		// If the version was not set via the command line, default to the same version as the application server
		appVersionParams := session_management.NewAppVersionParams().
			WithDefaults()
		appVersionResponse, err := ctx.Client.PlatformAPIs.SessionManagement.AppVersion(appVersionParams)
		if err != nil {
			return validateReleaseError(releases.GetPayload(), fmt.Errorf("unable to retrieve yugaware server version"))
		}

		if version, ok := appVersionResponse.GetPayload()["version"]; ok {
			o.Version = version
		} else {
			return validateReleaseError(releases.GetPayload(), fmt.Errorf("app version response did not contain version string"))
		}
	}

	for release := range releases.GetPayload() {
		if release == o.Version {
			ctx.Log.V(1).Info("found release version", "release", release)
			return nil
		}
	}

	return validateReleaseError(releases.GetPayload(), fmt.Errorf("could not find release"))
}

func (o *CreateOptions) getAccessKey(ctx *cmdutil.YWClientContext) error {
	// Kubernetes does not use an access key
	if o.provider.Code == "kubernetes" {
		return nil
	}

	ctx.Log.V(1).Info("getting access keys")
	params := access_keys.NewListParams().
		WithContext(ctx).
		WithCUUID(ctx.Client.CustomerUUID()).
		WithPUUID(o.provider.UUID)
	accessKeys, err := ctx.Client.PlatformAPIs.AccessKeys.List(params, ctx.Client.SwaggerAuth)
	if err != nil {
		return err
	}
	ctx.Log.V(1).Info("got access keys", "access_keys", accessKeys.GetPayload())
	for _, key := range accessKeys.GetPayload() {
		// TODO: should we always get the first key?
		o.accessKey = key.IDKey.KeyCode
		return nil
	}

	return fmt.Errorf("did not find access key")
}

func (o *CreateOptions) getRegionUUIDs() []strfmt.UUID {
	var regionUUIDs []strfmt.UUID
	for _, region := range o.regions {
		regionUUIDs = append(regionUUIDs, region.UUID)
	}
	return regionUUIDs
}
