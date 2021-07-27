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
	"fmt"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile                       string
	debug                         bool
	dialTimeout                   int64
	masterAddresses               string
	caCert, clientCert, clientKey string
	skipHostVerification          bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "yugatool",
	Short: "A tool to make troubleshooting yugabyte somewhat easier",
	Long:  `A tool to make troubleshooting yugabyte somewhat easier`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global configuration flags
	flags := rootCmd.PersistentFlags()
	flags.StringVar(&cfgFile, "config", "", "config file (default is $HOME/.yugatool.yaml)")
	flags.BoolVar(&debug, "debug", false, "debug mode")
	flags.StringVarP(&masterAddresses, "master-address", "m", "", "The master addresses")
	flags.Int64Var(&dialTimeout, "dialtimeout", 10, "number of seconds for dial timeouts")
	flags.BoolVar(&skipHostVerification, "skiphostverification", false, "skip ssl host verification")
	flags.StringVarP(&caCert, "cacert", "c", "", "the path to the CA certificate")
	flags.StringVar(&clientCert, "client-cert", "", "the path to the client certificate")
	flags.StringVar(&clientKey, "client-key", "", "the path to the client key file")

	if err := cobra.MarkFlagRequired(rootCmd.PersistentFlags(), "master-address"); err != nil {
		panic(err)
	}
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

		// Search config in home directory with name ".yugatool" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".yugatool")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
