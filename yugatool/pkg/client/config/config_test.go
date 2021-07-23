package config_test

import (
	"time"

	"github.com/blang/vfs"
	"github.com/blang/vfs/memfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client/config"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client/dial"
	"github.com/yugabyte/yb-tools/yugatool/pkg/test/util"
)

var _ = Describe("Config", func() {
	var (
		hosts          = []string{"master-1:7100,master2:7100,master3:7100"}
		universeConfig *config.UniverseConfig
		dialTimeout    = 30
	)
	BeforeEach(func() {
		universeConfig = &config.UniverseConfig{
			Fs:      vfs.OS(),
			Masters: hosts,
			Timeout: time.Duration(dialTimeout) * time.Second,
		}
	})
	Context("GetDialer", func() {
		var (
			dialer      dial.Dialer
			dialerError error
		)
		JustBeforeEach(func() {
			dialer, dialerError = universeConfig.GetDialer()
		})

		Context("NetDialer", func() {
			When("is not set", func() {
				It("returns a NetDialer", func() {
					Expect(dialer).To(BeEquivalentTo(&dial.NetDialer{Timeout: time.Duration(dialTimeout) * time.Second}))
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
				universeConfig.SslOpts = &config.SslOptions{
					SkipHostVerification: false,
					CaCertPath:           CaCertPath,
					CertPath:             CertPath,
					KeyPath:              KeyPath,
				}

				universeConfig.Fs = memfs.Create()
				CaCertPEM, err := util.GenerateCACertificate()
				Expect(err).NotTo(HaveOccurred())

				err = vfs.WriteFile(universeConfig.Fs, CaCertPath, CaCertPEM, 0600)
				Expect(err).NotTo(HaveOccurred())

				clientCert, clientKey, err := util.GeenerateClientCertFromCACertPEM(CaCertPEM)
				Expect(err).NotTo(HaveOccurred())

				err = vfs.WriteFile(universeConfig.Fs, CertPath, clientCert, 0600)
				Expect(err).NotTo(HaveOccurred())

				err = vfs.WriteFile(universeConfig.Fs, KeyPath, clientKey, 0600)
				Expect(err).NotTo(HaveOccurred())
			})
			When("all certs are present", func() {
				It("returns a TLSDialer", func() {
					Expect(dialerError).NotTo(HaveOccurred())
					dialarImpl := dialer.(*dial.TLSDialer)
					Expect(dialarImpl.Timeout).To(Equal(time.Duration(dialTimeout) * time.Second))
				})
			})
			When("the cacert is missing", func() {
				BeforeEach(func() {
					Expect(vfs.RemoveAll(universeConfig.Fs, CaCertPath)).To(Succeed())
				})
				It("returns an error", func() {
					Expect(dialerError).To(MatchError("open /ca.crt: file does not exist"))
				})
			})
			When("the clientCert is missing", func() {
				BeforeEach(func() {
					Expect(vfs.RemoveAll(universeConfig.Fs, CertPath)).To(Succeed())
				})
				It("returns an error", func() {
					Expect(dialerError).To(MatchError("unable to read x509 certificate: open /yugabyte.crt: file does not exist"))
				})
			})
			When("the client key is missing", func() {
				BeforeEach(func() {
					Expect(vfs.RemoveAll(universeConfig.Fs, KeyPath)).To(Succeed())
				})
				It("returns an error", func() {
					Expect(dialerError).To(MatchError("unable to read client key: open /yugabyte.key: file does not exist"))
				})
			})
			When("the cacert is corrupted", func() {
				BeforeEach(func() {
					Expect(vfs.WriteFile(universeConfig.Fs, CaCertPath, []byte{'a'}, 0600)).To(Succeed())
				})
				It("returns an error", func() {
					Expect(dialerError).To(MatchError("unable to add /ca.crt to the CA list"))
				})
			})
			When("the client certificate is corrupted", func() {
				BeforeEach(func() {
					Expect(vfs.WriteFile(universeConfig.Fs, CertPath, []byte{'a'}, 0600)).To(Succeed())
				})
				It("returns an error", func() {
					Expect(dialerError).To(MatchError("unable to read x509 key pair: tls: failed to find any PEM data in certificate input"))
				})
			})
			When("the client key is corrupted", func() {
				BeforeEach(func() {
					Expect(vfs.WriteFile(universeConfig.Fs, KeyPath, []byte{'a'}, 0600)).To(Succeed())
				})
				It("returns an error", func() {
					Expect(dialerError).To(MatchError("unable to read x509 key pair: tls: failed to find any PEM data in key input"))
				})
			})
			When("a client certificate is supplied without a correponding client key", func() {
				BeforeEach(func() {
					universeConfig.SslOpts.CertPath = ""
				})
				It("returns an error", func() {
					Expect(dialerError).To(MatchError("client certificate and key must both be set"))
				})
			})
			When("skiphostverification is set", func() {
				BeforeEach(func() {
					universeConfig.SslOpts = &config.SslOptions{SkipHostVerification: true}
				})
				It("returns a TLSDialer", func() {
					Expect(dialerError).NotTo(HaveOccurred())
					dialarImpl := dialer.(*dial.TLSDialer)
					Expect(dialarImpl.Timeout).To(Equal(time.Duration(dialTimeout) * time.Second))
				})
			})
		})
	})

})
