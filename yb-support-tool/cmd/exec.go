package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/yugabyte/yb-tools/yb-support-tool/exec"
)

var (
	command    string
	universe   string
	iFileName  string
	user       string
	isParallel bool
	hostname   string
	apiToken   string
	isInsecure bool
	isVerbose  bool
)

var inventories []exec.Inventory

var execCmd = &cobra.Command{
	Use:   "exec",
	Short: "Run commands across all nodes in a universe",
	Args:  cobra.OnlyValidArgs,

	// nolint: errcheck
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
		os.Exit(0)
	},
}

var ybaCmd = &cobra.Command{
	Use:   "yba",
	Short: "Lookup inventory using YBA API",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		inventories = exec.YbaLookup(hostname, apiToken, isInsecure, isVerbose)
		exec.SshCmd(inventories, universe, user, command, isParallel, isVerbose)
	},
}

var fileCmd = &cobra.Command{
	Use:   "file",
	Short: "Lookup inventory using inventory file",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		inventories = exec.FileLookup(iFileName, isVerbose)
		exec.SshCmd(inventories, universe, user, command, isParallel, isVerbose)
	},
}

// nolint: errcheck
func addExecCmdsFlags() {

	// global flags
	execCmd.PersistentFlags().StringVar(&command, "cmd", "", "command to run across all universe nodes")
	execCmd.PersistentFlags().StringVar(&universe, "universe", "", "universe to run commands against")
	execCmd.PersistentFlags().BoolVarP(&isParallel, "parallel", "p", false, "run commands against all nodes simultaneously")
	execCmd.PersistentFlags().StringVarP(&user, "user", "u", "yugabyte", "ssh user to run commands with")
	execCmd.PersistentFlags().BoolVar(&isVerbose, "verbose", false, "show node details next to ssh output")

	execCmd.MarkPersistentFlagRequired("cmd")

	// yba command
	execCmd.AddCommand(ybaCmd)

	ybaCmd.Flags().StringVar(&hostname, "hostname", "", "YBA hostname or IP")
	ybaCmd.Flags().StringVar(&apiToken, "api-token", "", "API token for YBA authentication")
	ybaCmd.Flags().BoolVarP(&isInsecure, "insecure", "k", false, "Allow self-signed certificates")

	ybaCmd.MarkFlagRequired("hostname")
	ybaCmd.MarkFlagRequired("api-token")

	// file command
	execCmd.AddCommand(fileCmd)

	fileCmd.Flags().StringVarP(&iFileName, "inventory", "i", "", "file containing universe inventory")
	fileCmd.MarkFlagRequired("inventory")

}
