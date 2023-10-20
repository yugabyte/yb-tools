package sendsafelyuploader

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"golang.org/x/crypto/pbkdf2"
)

func createClientSecret() string {

	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		panic(err)

	}
	return base64.RawURLEncoding.EncodeToString(token)

}

func (p *Package) createChecksum() {
	dk := pbkdf2.Key([]byte(p.Uploader.ClientSecret), []byte(p.Info.PackageCode), 1024, 32, sha256.New)
	p.Checksum = hex.EncodeToString(dk)
}

func encrypt(passphrase []byte, message io.Reader) (*bytes.Buffer, error) {

	// configuration for file encryption
	var encryptConfig packet.Config
	encryptConfig.DefaultCipher = packet.CipherAES256
	encryptConfig.DefaultCompressionAlgo = packet.CompressionNone
	encryptConfig.DefaultHash = crypto.SHA256

	buf := new(bytes.Buffer)

	// Create encryptor
	encryptorWriter, err := openpgp.SymmetricallyEncrypt(buf, passphrase, &openpgp.FileHints{IsBinary: true}, &encryptConfig)
	if err != nil {
		return nil, fmt.Errorf("Error creating entity for encryption: %v", err)
	}

	_, err = io.Copy(encryptorWriter, message)

	if err != nil {
		return nil, fmt.Errorf("Error writing data to be encrypted: %v", err)
	}

	encryptorWriter.Close()

	// Return buffer output - an encrypted message
	return buf, nil
}

func (p *Package) Encrypt(unencryptedFilePart io.Reader) (*bytes.Buffer, error) {
	passphrase := p.Info.ServerSecret + p.Uploader.ClientSecret
	return encrypt([]byte(passphrase), unencryptedFilePart)
}
