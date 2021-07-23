package util

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"

	"github.com/pkg/errors"
)

func GenerateCACertificate() ([]byte, error) {
	certTemplate := &x509.Certificate{
		Version:      3,
		SerialNumber: big.NewInt(1),
		Issuer: pkix.Name{
			CommonName: "Yugabyte DB",
		},
		NotBefore: time.Now().Add(-1 * time.Minute),
		NotAfter:  time.Now().AddDate(10, 0, 0),
		Subject: pkix.Name{
			CommonName: "Yugabyte DB",
		},
		KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
		BasicConstraintsValid: true,
		IsCA:                  true,
		IPAddresses:           nil,
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return []byte{}, err
	}

	rootCACert, err := x509.CreateCertificate(rand.Reader, certTemplate, certTemplate, &key.PublicKey, key)
	if err != nil {
		return []byte{}, err
	}

	rootCACertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootCACert})

	return rootCACertPEM, nil
}

func GeenerateClientCertFromCACertPEM(CACertPEM []byte) (cert, privatekey []byte, err error) {
	CACertBlock, restOfPEM := pem.Decode(CACertPEM)
	if len(restOfPEM) > 0 {
		err = errors.New("extra bytes left over from decoding CACert")
		return
	}

	cacert, err := x509.ParseCertificate(CACertBlock.Bytes)
	if err != nil {
		return
	}

	return GenerateClientCertificate(cacert)
}

func GenerateClientCertificate(CACert *x509.Certificate) (cert, privateKey []byte, err error) {
	certTemplate := &x509.Certificate{
		Version:      3,
		SerialNumber: big.NewInt(2),
		NotBefore:    time.Now().Add(-1 * time.Minute),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		Subject: pkix.Name{
			CommonName: "yb-tserver-0.yb-tservers.yugabyte-db.svc.cluster.local",
		},
		KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
		DNSNames: []string{"*.*.yugabyte-db", "*.*.yugabyte-db.svc.cluster.local"},
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, certTemplate, CACert, &key.PublicKey, key)
	if err != nil {
		return
	}

	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return
	}
	cert = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	privateKey = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privateKeyBytes})
	return
}
