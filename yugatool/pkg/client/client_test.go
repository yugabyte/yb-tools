package client_test

import (
	"github.com/blang/vfs"
	"github.com/blang/vfs/memfs"
	. "github.com/icza/gox/gox"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/common"
	"github.com/yugabyte/yb-tools/yugatool/api/yugatool/config"
	ybclient "github.com/yugabyte/yb-tools/yugatool/pkg/client"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client/dial"
	"github.com/yugabyte/yb-tools/yugatool/pkg/test/util"
)

var _ = Describe("Client", func() {
	var (
		hosts = []*common.HostPortPB{
			{Host: NewString("master-1"), Port: NewUint32(7100)},
			{Host: NewString("master-2"), Port: NewUint32(7100)},
			{Host: NewString("master-3"), Port: NewUint32(7100)},
		}
		yugabyteClient *ybclient.YBClient
		dialTimeout    int64 = 30
	)
	BeforeEach(func() {
		yugabyteClient = &ybclient.YBClient{
			//Log:     logr.Logger{},
			Fs: memfs.Create(),
			//Context: nil,
			Config: &config.UniverseConfigPB{
				Masters:        hosts,
				TimeoutSeconds: NewInt64(dialTimeout),
			},
		}
	})
	Context("GetDialer", func() {
		var (
			dialer      dial.Dialer
			dialerError error
		)
		JustBeforeEach(func() {
			dialer, dialerError = yugabyteClient.GetDialer()
		})

		Context("NetDialer", func() {
			When("SSlOpts is not set", func() {
				It("returns a NetDialer", func() {
					Expect(dialer).To(BeEquivalentTo(&dial.NetDialer{TimeoutSeconds: dialTimeout}))
					Expect(dialerError).NotTo(HaveOccurred())
				})
			})
			When("SSlOpts is the default", func() {
				BeforeEach(func() {
					yugabyteClient.Config.SslOpts = &config.SslOptionsPB{
						SkipHostVerification: NewBool(false),
						CaCertPath:           NewString(""),
						CertPath:             NewString(""),
						KeyPath:              NewString(""),
					}
				})
				It("returns a NetDialer", func() {
					Expect(dialer).To(BeEquivalentTo(&dial.NetDialer{TimeoutSeconds: dialTimeout}))
					Expect(dialerError).NotTo(HaveOccurred())
				})
			})
		})

		Context("SSLDialer", func() {
			var (
				CaCertPath = "/ca.crt"
				CertPath   = "/yugabyte.crt"
				KeyPath    = "/yugabyte.key"
			)
			BeforeEach(func() {
				yugabyteClient.Config.SslOpts = &config.SslOptionsPB{
					SkipHostVerification: NewBool(false),
					CaCertPath:           NewString(CaCertPath),
					CertPath:             NewString(CertPath),
					KeyPath:              NewString(KeyPath),
				}

				yugabyteClient.Fs = memfs.Create()
				CaCertPEM, err := util.GenerateCACertificate()
				Expect(err).NotTo(HaveOccurred())

				err = vfs.WriteFile(yugabyteClient.Fs, CaCertPath, CaCertPEM, 0600)
				Expect(err).NotTo(HaveOccurred())

				clientCert, clientKey, err := util.GeenerateClientCertFromCACertPEM(CaCertPEM)
				Expect(err).NotTo(HaveOccurred())

				err = vfs.WriteFile(yugabyteClient.Fs, CertPath, clientCert, 0600)
				Expect(err).NotTo(HaveOccurred())

				err = vfs.WriteFile(yugabyteClient.Fs, KeyPath, clientKey, 0600)
				Expect(err).NotTo(HaveOccurred())
			})
			When("all certs are present", func() {
				It("returns a TLSDialer", func() {
					Expect(dialerError).NotTo(HaveOccurred())
					dialarImpl := dialer.(*dial.TLSDialer)
					Expect(dialarImpl.TimeoutSeconds).To(Equal(dialTimeout))
				})
			})
			When("the cacert is missing", func() {
				BeforeEach(func() {
					Expect(vfs.RemoveAll(yugabyteClient.Fs, CaCertPath)).To(Succeed())
				})
				It("returns an error", func() {
					Expect(dialerError).To(MatchError("open /ca.crt: file does not exist"))
				})
			})
			When("the clientCert is missing", func() {
				BeforeEach(func() {
					Expect(vfs.RemoveAll(yugabyteClient.Fs, CertPath)).To(Succeed())
				})
				It("returns an error", func() {
					Expect(dialerError).To(MatchError("unable to read x509 certificate: open /yugabyte.crt: file does not exist"))
				})
			})
			When("the client key is missing", func() {
				BeforeEach(func() {
					Expect(vfs.RemoveAll(yugabyteClient.Fs, KeyPath)).To(Succeed())
				})
				It("returns an error", func() {
					Expect(dialerError).To(MatchError("unable to read client key: open /yugabyte.key: file does not exist"))
				})
			})
			When("the cacert is corrupted", func() {
				BeforeEach(func() {
					Expect(vfs.WriteFile(yugabyteClient.Fs, CaCertPath, []byte{'a'}, 0600)).To(Succeed())
				})
				It("returns an error", func() {
					Expect(dialerError).To(MatchError("unable to add /ca.crt to the CA list"))
				})
			})
			When("the client certificate is corrupted", func() {
				BeforeEach(func() {
					Expect(vfs.WriteFile(yugabyteClient.Fs, CertPath, []byte{'a'}, 0600)).To(Succeed())
				})
				It("returns an error", func() {
					Expect(dialerError).To(MatchError("unable to read x509 key pair: tls: failed to find any PEM data in certificate input"))
				})
			})
			When("the client key is corrupted", func() {
				BeforeEach(func() {
					Expect(vfs.WriteFile(yugabyteClient.Fs, KeyPath, []byte{'a'}, 0600)).To(Succeed())
				})
				It("returns an error", func() {
					Expect(dialerError).To(MatchError("unable to read x509 key pair: tls: failed to find any PEM data in key input"))
				})
			})
			When("a client certificate is supplied without a correponding client key", func() {
				BeforeEach(func() {
					yugabyteClient.Config.GetSslOpts().CertPath = nil
				})
				It("returns an error", func() {
					Expect(dialerError).To(MatchError("client certificate and key must both be set"))
				})
			})
			When("skiphostverification is set", func() {
				BeforeEach(func() {
					yugabyteClient.Config.SslOpts = &config.SslOptionsPB{SkipHostVerification: NewBool(true)}
				})
				It("returns a TLSDialer", func() {
					Expect(dialerError).NotTo(HaveOccurred())
					dialarImpl := dialer.(*dial.TLSDialer)
					Expect(dialarImpl.TimeoutSeconds).To(Equal(dialTimeout))
				})
			})
		})
	})
})
