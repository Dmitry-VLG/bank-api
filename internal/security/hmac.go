package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func ComputeHMAC(data string, secret []byte) string {
	h := hmac.New(sha256.New, secret)
	_, _ = h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func VerifyHMAC(data, expected string, secret []byte) bool {
	got := ComputeHMAC(data, secret)
	return hmac.Equal([]byte(got), []byte(expected))
}
