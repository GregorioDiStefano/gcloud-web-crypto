package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func (c *CryptoData) GenerateHMAC(data []byte) string {
	sig := hmac.New(sha256.New, c.HMACSecret)
	sig.Write(data)

	return hex.EncodeToString(sig.Sum(nil))
}
