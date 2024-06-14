package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Version = "pre-release"

var (
	rootCmd = &cobra.Command{
		Use:   "yb-support-tool",
		Short: "Yugabyte Support Tool",
	}

	versionCmd = &cobra.Command{
		Use: "version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(Version)
			os.Exit(0)
		},
	}
)

func Execute() {

	if err := rootCmd.Execute(); err != nil {
		_, err := fmt.Fprintln(os.Stderr, err)
		if err != nil {
			return
		}
		os.Exit(1)
	}
}

// nolint: errcheck
func init() {

	// import existing tools from yb-tools repo
	importYbTools()

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(uploadCmd)
	rootCmd.AddCommand(execCmd)
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Print verbose logs")

	addUploadFlags()
	addExecCmdsFlags()
}
