package exec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/blang/vfs"
	ywcCmd "github.com/yugabyte/yb-tools/yugaware-client/cmd"
	"gopkg.in/yaml.v2"
)

type Inventory struct {
	Name            string `json:"name" yaml:"name"`
	UniverseDetails struct {
		Clusters []struct {
			ClusterType string `json:"clusterType" yaml:"clusterType"`
			UserIntent  struct {
				AccessKeyCode string `json:"accessKeyCode" yaml:"accessKeyCode"`
				Provider      string `json:"provider" yaml:"provider"`
			} `json:"userIntent" yaml:"userIntent"`
		} `json:"clusters" yaml:"clusters"`
		NodeDetailsSet []struct {
			CloudInfo struct {
				PrivateIP string `json:"private_ip" yaml:"private_ip"`
			} `json:"cloudInfo" yaml:"cloudInfo"`
			NodeName string `json:"nodeName" yaml:"nodeName"`
			NodeIdx  int    `json:"nodeIdx" yaml:"nodeIdx"`
			IsMaster bool   `json:"isMaster" yaml:"isMaster"`
		} `json:"nodeDetailsSet" yaml:"nodeDetailsSet"`
		NodePrefix string `json:"nodePrefix" yaml:"nodePrefix"`
	} `json:"universeDetails" yaml:"universeDetails"`
}

func YbaLookup(hostname, apiToken string, isInsecure, isVerbose bool) []Inventory {

	// yugaware-client returns the API response wrapped in "content: "
	type apiResp struct {
		Content []Inventory `json:"content"`
	}

	if isVerbose {
		fmt.Println("Initializing yugaware-client")
	}

	var fs vfs.Filesystem
	ywCommand := ywcCmd.RootInit(fs)

	var (
		content apiResp
		args    []string
	)

	if isVerbose {
		fmt.Printf("Command: \"yugaware-client universe list -o json --hostname %s --api-token <redacted>\"\n", hostname)
	}

	args = append(args, "universe", "list", "-o", "json")
	args = append(args, "--hostname", hostname, "--api-token", apiToken)
	if isInsecure {
		args = append(args, "--skiphostverification")
	}
	buf := new(bytes.Buffer)
	ywCommand.SetOut(buf)
	ywCommand.SetErr(buf)

	ywCommand.SetArgs(args)

	// run yugaware-client command using cobra
	err := ywCommand.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	buffer, err := buf.Bytes(), err

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = json.Unmarshal(buffer, &content)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// content.Content is the raw api response, which is "[]inventory"
	return content.Content
}

func FileLookup(iFileName string, isVerbose bool) []Inventory {

	var inventories []Inventory

	if isVerbose {
		fmt.Printf("Opening inventory file: \"%s\"\n", iFileName)
	}

	f, _ := filepath.Abs(iFileName)
	inventoryFile, err := os.Open(f)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer inventoryFile.Close()

	inventoryBytes, err := io.ReadAll(inventoryFile)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// this will unmarshal both json and yaml properly
	err = yaml.Unmarshal(inventoryBytes, &inventories)

	if err != nil {
		fmt.Printf("Unable to parse inventory file \"%s\": %s", iFileName, err)
		os.Exit(1)
	}

	return inventories
}
