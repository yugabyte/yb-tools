package universe

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yugabyte/yb-tools/pkg/format"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/universe_information"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/cmdutil"
)

func HealthCheckCmd(ctx *cmdutil.YWClientContext) *cobra.Command {
	options := &HealthCheckOptions{}
	cmd := &cobra.Command{
		Use:   "healthcheck (UNIVERSE_NAME|UNIVERSE_UUID)",
		Short: "Show universe healthchecks",
		Long:  `Show universe healthchecks`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := ctx.WithCmd(cmd).WithOptions(options).Setup()
			if err != nil {
				return err
			}

			// Positional argument
			universeIdentifier := args[0]

			return healthCheckCmd(ctx, universeIdentifier, options.ErrorsOnly)
		},
	}
	options.AddFlags(cmd)

	return cmd
}

type HealthCheckOptions struct {
	ErrorsOnly bool `mapstructure:"errors_only"`
}

func (h *HealthCheckOptions) AddFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&h.ErrorsOnly, "errors-only", false, "only show health checks that failed")
}

func (h *HealthCheckOptions) Validate(_ *cmdutil.YWClientContext) error {
	return nil
}

var _ cmdutil.CommandOptions = &HealthCheckOptions{}

func healthCheckCmd(ctx *cmdutil.YWClientContext, universeIdentifier string, errorsOnly bool) error {
	universe, err := ctx.Client.GetUniverseByIdentifier(universeIdentifier)
	if err != nil {
		return err
	}
	if universe == nil {
		return fmt.Errorf("universe does not exist: %s", universeIdentifier)
	}

	params := universe_information.NewHealthCheckUniverseParams().WithDefaults().
		WithCUUID(ctx.Client.CustomerUUID()).
		WithUniUUID(universe.UniverseUUID)

	healthcheck, err := ctx.Client.PlatformAPIs.UniverseInformation.HealthCheckUniverse(params, ctx.Client.SwaggerAuth)
	if err != nil {
		return err
	}

	type Healthcheck struct {
		Data []struct {
			Details   []string `json:"details"`
			HasError  bool     `json:"has_error"`
			Message   string   `json:"message"`
			Node      string   `json:"node"`
			NodeName  string   `json:"node_name"`
			Process   string   `json:"process,omitempty"`
			Timestamp string   `json:"timestamp"`
		} `json:"data"`
		HasError  bool   `json:"has_error"`
		Timestamp string `json:"timestamp"`
		YbVersion string `json:"yb_version"`
	}

	healtheckBytes, err := json.Marshal(healthcheck.GetPayload())
	if err != nil {
		return err
	}

	var healthchecks = []Healthcheck{}
	err = json.Unmarshal(healtheckBytes, &healthchecks)
	if err != nil {
		return err
	}

	for _, hc := range healthchecks {
		table := &format.Output{
			OutputMessage: fmt.Sprintf("%s Health Check %s", universeIdentifier, hc.Timestamp),
			JSONObject:    hc.Data,
			OutputType:    ctx.GlobalOptions.Output,
			TableColumns: []format.Column{
				{Name: "TYPE", JSONPath: "$.message"},
				{Name: "NODE", JSONPath: "$.node"},
				{Name: "NODE_NAME", JSONPath: "$.node_name"},
				{Name: "ERROR", JSONPath: "$.has_error"},
				{Name: "TIMESTAMP", JSONPath: "$.timestamp"},
			},
		}

		if errorsOnly {
			if !hc.HasError {
				continue
			}
			table.Filter = "@.has_error == true"
		}

		err = table.Println()
		if err != nil {
			return err
		}
	}
	return nil
}
