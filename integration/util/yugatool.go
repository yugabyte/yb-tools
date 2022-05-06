package util

import (
	"bytes"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/blang/vfs"
	"github.com/blang/vfs/memfs"
	"github.com/go-logr/logr"
	. "github.com/onsi/gomega"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/common"
	"github.com/yugabyte/yb-tools/yugatool/api/yugatool/config"
	"github.com/yugabyte/yb-tools/yugatool/cmd"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client"
	"github.com/yugabyte/yb-tools/yugatool/pkg/util"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
)

type YugatoolContext struct {
	*client.YBClient

	Output               string
	DialTimeout          int64
	MasterAddresses      []*common.HostPortPB
	CACert               string
	ClientCert           string
	ClientKey            string
	SkipHostVerification bool

	UniverseInfo *models.UniverseResp

	Fs vfs.Filesystem
}

func NewYugatoolContext(logger logr.Logger, universe *models.UniverseResp, masters []*common.HostPortPB, dialTimeout int64, cacert, clientCert, clientKey []byte, skipHostVerification bool) *YugatoolContext {
	var caCertPath, clientCertPath, clientKeyPath string
	fs := memfs.Create()

	if cacert != nil {
		caCertPath = "/cacert.pem"
		err := vfs.WriteFile(fs, caCertPath, cacert, 0600)
		Expect(err).NotTo(HaveOccurred())
	}

	if clientCert != nil {
		clientCertPath = "/client_cert.pem"
		err := vfs.WriteFile(fs, clientCertPath, cacert, 0600)
		Expect(err).NotTo(HaveOccurred())
	}

	if clientKey != nil {
		clientKeyPath = "/client_cert.pem"
		err := vfs.WriteFile(fs, clientKeyPath, cacert, 0600)
		Expect(err).NotTo(HaveOccurred())
	}

	c := &client.YBClient{
		Log: logger.WithName("test"),
		Fs:  fs,
		Config: &config.UniverseConfigPB{
			Masters:        masters,
			TimeoutSeconds: &dialTimeout,
			TlsOpts: &config.TlsOptionsPB{
				SkipHostVerification: &skipHostVerification,
				CaCertPath:           &caCertPath,
				CertPath:             &clientCertPath,
				KeyPath:              &clientKeyPath,
			},
		},
	}

	err := c.Connect()

	Expect(err).NotTo(HaveOccurred())

	return &YugatoolContext{
		YBClient: c,

		DialTimeout:          dialTimeout,
		MasterAddresses:      masters,
		CACert:               caCertPath,
		ClientCert:           clientCertPath,
		ClientKey:            clientKeyPath,
		SkipHostVerification: skipHostVerification,

		UniverseInfo: universe,

		Fs: fs,
	}
}

func (c *YugatoolContext) mastersString() string {
	masters := strings.Builder{}

	for i, m := range c.MasterAddresses {
		master := util.HostPortString(m)

		masters.WriteString(master)
		if i < len(c.MasterAddresses)-1 {
			masters.WriteRune(',')
		}
	}
	return masters.String()
}

func (c *YugatoolContext) RunYugatoolCommand(args ...string) (*bytes.Buffer, error) {
	ytCommand := cmd.RootInit(c.Fs)

	args = append(args, "-m", c.mastersString(), "--dial-timeout", strconv.Itoa(int(c.DialTimeout)))

	if c.SkipHostVerification {
		args = append(args, "--skiphostverification")
	}

	if c.CACert != "" {
		args = append(args, "--cacert", c.CACert)
	}

	if c.ClientCert != "" {
		args = append(args, "--client-cert", c.ClientCert)
	}

	if c.ClientKey != "" {
		args = append(args, "--client-key", c.ClientKey)
	}

	buf := new(bytes.Buffer)
	ytCommand.SetOut(buf)
	ytCommand.SetErr(buf)

	ytCommand.SetArgs(args)

	err := ytCommand.Execute()

	return buf, err
}

func (c *YugatoolContext) YSQLConnection() *sql.DB {
	var psqlInfo string
	for _, server := range c.UniverseInfo.UniverseDetails.NodeDetailsSet {
		if server.IsTserver {
			sslMode := "disable"
			if c.UniverseInfo.UniverseDetails.Clusters[0].UserIntent.EnableClientToNodeEncrypt {
				sslMode = "require"
			}

			psqlInfo = fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s",
				server.CloudInfo.PrivateIP, server.YsqlServerRPCPort, "yugabyte", "yugabyte", sslMode)
			break
		}
	}
	db, err := sql.Open("postgres", psqlInfo)
	Expect(err).NotTo(HaveOccurred())
	return db
}
