package main

import (
	"errors"
	"io"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"

	gc "github.com/GregorioDiStefano/gcloud-web-crypto"
)

var packetConfig packet.Config

func init() {
	packetConfig = packet.Config{
		DefaultCipher:          packet.CipherAES256,
		DefaultCompressionAlgo: packet.CompressionZLIB,
		CompressionConfig:      &packet.CompressionConfig{Level: packet.BestSpeed},
	}
}

func Encrypt(src io.Reader, w io.Writer) (err error) {
	password := gc.PasswordConf.PgpPassword
	cipherText, err := openpgp.SymmetricallyEncrypt(w, password, nil, &packetConfig)

	if err != nil {
		return
	}
	defer cipherText.Close()

	if err != nil {
		return
	}

	writer := io.MultiWriter(cipherText)
	_, err = io.Copy(writer, src)

	if err != nil {
		return
	}
	return
}

func Decrypt(r io.Reader, df io.Writer) (err error) {
	password := gc.PasswordConf.PgpPassword

	failed := false
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

	io.Copy(df, md.UnverifiedBody)
	return
}
