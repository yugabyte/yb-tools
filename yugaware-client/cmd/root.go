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

package cmd

import (
	"os"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yugabyte/yb-tools/yugaware-client/cmd/provider"
	"github.com/yugabyte/yb-tools/yugaware-client/cmd/universe"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/cmdutil"
)

var (
	cfgFile string

	Version = "DEV"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = RootInit()

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".ywclient" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".ywclient")
	}

	viper.SetEnvPrefix("YW")
	viper.AutomaticEnv() // read in environment variables that match
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// If a config file is found, read it in.
	_ = viper.ReadInConfig()
}

func RootInit() *cobra.Command {
	globalOptions := &cmdutil.YWGlobalOptions{}

	cmd := &cobra.Command{
		Use:     "yugaware-client",
		Short:   `A tool to deploy/configure yugaware`,
		Long:    `A tool to deploy/configure yugaware`,
		Version: Version,
	}

	cmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.yugatool.yaml)")
	globalOptions.AddFlags(cmd)

	ctx := cmdutil.NewCommandContext().
		WithGlobalOptions(globalOptions)

	// Top level commands
	cmd.AddCommand(RegisterCmd(ctx))
	cmd.AddCommand(LoginCmd(ctx))

	type CommandCategory struct {
		Name        string
		Description string
		Commands    []*cobra.Command
	}

	categories := []CommandCategory{
		{
			Name:        "provider",
			Description: "Interact with Yugaware providers",
			Commands: []*cobra.Command{
				provider.CreateCmd(ctx),
				provider.ListCmd(ctx),
			},
		},
		{
			Name:        "universe",
			Description: "Interact with Yugabyte universes",
			Commands: []*cobra.Command{
				universe.CreateUniverseCmd(ctx),
				universe.DeleteUniverseCmd(ctx),
				universe.ListCmd(ctx),
			},
		},
	}

	for _, category := range categories {
		categoryCmd := &cobra.Command{
			Use:   category.Name,
			Short: category.Description,
			Long:  category.Description,
			Run:   cmd.HelpFunc(),
		}
		for _, subcommand := range category.Commands {
			categoryCmd.AddCommand(subcommand)
		}
		cmd.AddCommand(categoryCmd)
	}

	return cmd
}
