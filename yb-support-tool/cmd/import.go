package cmd

import (
	"github.com/blang/vfs"
	ytCmd "github.com/yugabyte/yb-tools/yugatool/cmd"
	ywcCmd "github.com/yugabyte/yb-tools/yugaware-client/cmd"
)

func importYbTools() {

	// needed for ywc
	var fs vfs.Filesystem

	// import as type "cobra.Command" from github repos
	importYtCmd := ytCmd.RootInit(vfs.OS())
	importYwcCmd := ywcCmd.RootInit(fs)

	// add new cobra commands to our root cobra commands
	rootCmd.AddCommand(importYtCmd)
	rootCmd.AddCommand(importYwcCmd)
}
