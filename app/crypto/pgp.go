package crypto

import (
	"crypto"
	"encoding/base64"
	"errors"
	"io"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"
)

var packetConfig packet.Config

func init() {
	packetConfig = packet.Config{
		DefaultHash:            crypto.SHA256,
		DefaultCipher:          packet.CipherAES256,
		DefaultCompressionAlgo: packet.CompressionZIP,
		CompressionConfig: &packet.CompressionConfig{
			Level: packet.BestSpeed,
		},
	}

	log.SetLevel(log.DebugLevel)
}

func (c *CryptoData) EncryptFile(src io.Reader, w io.Writer) (written int64, err error) {
	password := c.SymmetricKey
	log.WithFields(log.Fields{"key": base64.StdEncoding.EncodeToString(c.SymmetricKey)}).Debug("encrypting")
	cipherText, err := openpgp.SymmetricallyEncrypt(w, password, nil, &packetConfig)

	if err != nil {
		return
	}
	defer cipherText.Close()

	if err != nil {
		return
	}

	written, err = io.Copy(cipherText, src)

	if err != nil {
		return
	}
	return
}

func (c *CryptoData) DecryptFile(r io.Reader, w io.Writer) (err error) {
	password := c.SymmetricKey
	failed := false

	log.WithFields(log.Fields{"key": base64.StdEncoding.EncodeToString(c.SymmetricKey)}).Debug("decrypting")
	prompt := func(keys []openpgp.Key, symmetric bool) ([]byte, error) {
		// If the given passphrase isn't correct, the function will be called again, forever.
		if failed {
			return nil, errors.New("decryption failed")
		}
		failed = true
		return password, nil
	}

	md, err := openpgp.ReadMessage(r, nil, prompt, &packetConfig)

	if err != nil {
		return
	}

	io.Copy(w, md.UnverifiedBody)
	return
}
