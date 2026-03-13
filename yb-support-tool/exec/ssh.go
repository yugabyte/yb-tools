package exec

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
)

func runSsh(conn *ssh.Client, cmd, nodeIdx, nodeName, nodePrivateIp string, isVerbose bool) []byte {

	var res []byte

	session, err := conn.NewSession()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer session.Close()
	res, err = session.CombinedOutput(cmd)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	output := string(res)

	// by default, only the output is shown with no additional details
	// if the verbose flag is used, then node information is printed before the output of each command
	if isVerbose {

		// remove the last newline (if applicable) so we can accurately determine if this is multi-line output
		output = strings.TrimSuffix(output, "\n")

		newlineCount := len(strings.Split(output, "\n"))
		verboseFmt := "n" + nodeIdx + ": " + nodeName + ": " + nodePrivateIp

		if newlineCount > 1 {
			// print output on a separate line if it's multi-line output
			fmt.Println("-- " + verboseFmt + " --")
			fmt.Println(output)
		} else {
			// print everything on a single line if the output is single-line
			fmt.Println(verboseFmt + ": " + output)
		}
	} else {
		fmt.Print(output)
	}

	return res
}

func SshCmd(inventories []Inventory, universe, user, cmd string, isParallel bool, isVerbose bool) {

	var privateKeyPathPrefix = "/opt/yugabyte/yugaware/data/keys/"
	var privateKeyPath string

	for _, inventory := range inventories {
		for _, cluster := range inventory.UniverseDetails.Clusters {
			if cluster.ClusterType == "PRIMARY" {
				privateKeyPath = privateKeyPathPrefix + cluster.UserIntent.Provider + "/" + cluster.UserIntent.AccessKeyCode + ".pem"
				break
			}
		}
	}

	pemBytes, err := os.ReadFile(privateKeyPath)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	signer, err := ssh.ParsePrivateKey(pemBytes)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.HostKeyCallback(
			func(host string, remote net.Addr, key ssh.PublicKey) error {
				return nil
			},
		),
	}

	for _, inventory := range inventories {

		// was the universe flag used by the user?
		if universe != "" {

			// check to see if the universe name in this inventory matches the universe supplied by the user
			if inventory.Name != universe {
				continue
			}
		}

		// workgroup to use if isParallel is true
		wg := &sync.WaitGroup{}

		for _, nodeDetailsSet := range inventory.UniverseDetails.NodeDetailsSet {

			nodeIdx := strconv.Itoa(nodeDetailsSet.NodeIdx)
			nodeName := nodeDetailsSet.NodeName
			nodePrivateIp := nodeDetailsSet.CloudInfo.PrivateIP

			host := nodeDetailsSet.CloudInfo.PrivateIP + ":22"
			conn, err := ssh.Dial("tcp", host, config)

			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			if isParallel {
				wg.Add(1)
				go func(wg *sync.WaitGroup) {
					runSsh(conn, cmd, nodeIdx, nodeName, nodePrivateIp, isVerbose)
					wg.Done()
				}(wg)
			} else {
				runSsh(conn, cmd, nodeIdx, nodeName, nodePrivateIp, isVerbose)
			}
		}
		wg.Wait()
	}

}
