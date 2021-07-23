package util_test

import (
	"crypto/x509"
	"encoding/pem"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	util2 "github.com/yugabyte/yb-tools/yugatool/pkg/test/util"
)

var _ = Describe("Certificate", func() {
	var (
		cacertPEM, restOfPEM, clientCertPEM, clientKeyPEM []byte

		decodedPEM         *pem.Block
		CACert, clientCert *x509.Certificate
		//clientKeyInterface interface{}
		//clientKey          *rsa.PrivateKey

		certError       error
		parseError      error
		clientCertError error
	)
	BeforeEach(func() {
		cacertPEM, certError = util2.GenerateCACertificate()
		decodedPEM, restOfPEM = pem.Decode(cacertPEM)
		CACert, parseError = x509.ParseCertificate(decodedPEM.Bytes)
	})
	Context("GenerateCACertificate", func() {
		When("a CA cert is generated", func() {
			It("generates a valid certificate", func() {
				Expect(certError).NotTo(HaveOccurred())
				Expect(cacertPEM).Should(ContainSubstring("-----BEGIN CERTIFICATE-----"))
				Expect(cacertPEM).Should(ContainSubstring("-----END CERTIFICATE-----"))

				Expect(restOfPEM).To(HaveLen(0))

				Expect(parseError).NotTo(HaveOccurred())
				Expect(CACert.IsCA).To(BeTrue())
				Expect(CACert.KeyUsage).To(Equal(x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign))
				Expect(CACert.ExtKeyUsage).To(ContainElements(x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth))
				Expect(CACert.Issuer.CommonName).To(Equal("Yugabyte DB"))
				Expect(CACert.Subject.CommonName).To(Equal("Yugabyte DB"))
			})
		})
	})
	Context("GenerateClientCertificate", func() {
		BeforeEach(func() {
			clientCertPEM, clientKeyPEM, clientCertError = util2.GenerateClientCertificate(CACert)
		})
		When("generating a client certificate", func() {
			BeforeEach(func() {
				decodedPEM, restOfPEM = pem.Decode(clientCertPEM)
				clientCert, parseError = x509.ParseCertificate(decodedPEM.Bytes)
			})
			It("creates a valid client certificate", func() {
				Expect(clientCertError).NotTo(HaveOccurred())
				Expect(clientCertPEM).Should(ContainSubstring("-----BEGIN CERTIFICATE-----"))
				Expect(clientCertPEM).Should(ContainSubstring("-----END CERTIFICATE-----"))

				Expect(restOfPEM).To(HaveLen(0))

				Expect(parseError).NotTo(HaveOccurred())
				Expect(clientCert.IsCA).To(BeFalse())
				Expect(clientCert.KeyUsage).To(Equal(x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment))
				Expect(clientCert.ExtKeyUsage).To(ContainElements(x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth))
				Expect(clientCert.Issuer.CommonName).To(Equal("Yugabyte DB"))
				Expect(clientCert.Subject.CommonName).To(Equal("yb-tserver-0.yb-tservers.yugabyte-db.svc.cluster.local"))
			})
		})
		When("generating a client key", func() {
			BeforeEach(func() {
				decodedPEM, restOfPEM = pem.Decode(clientKeyPEM)
				_, parseError = x509.ParsePKCS8PrivateKey(decodedPEM.Bytes)
			})
		})
		It("creates a valid client key", func() {
			Expect(clientCertError).NotTo(HaveOccurred())
			Expect(clientKeyPEM).Should(ContainSubstring("-----BEGIN RSA PRIVATE KEY-----"))
			Expect(clientKeyPEM).Should(ContainSubstring("-----END RSA PRIVATE KEY-----"))

			Expect(restOfPEM).To(HaveLen(0))

			Expect(parseError).NotTo(HaveOccurred())

			//clientKey = clientKeyInterface.(*rsa.PrivateKey)
			//Expect(clientKey.Validate()).To(Succeed())
		})

	})
})
