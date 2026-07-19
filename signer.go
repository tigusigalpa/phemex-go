package phemex

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// Signer generates HMAC SHA256 request signatures for the Phemex API.
//
// The signing material is constructed according to the official Phemex
// documentation: the URL path (including the leading slash), the raw query
// string (without the leading question mark), the request expiry timestamp in
// seconds, and the request body for POST/PUT requests.
type Signer struct {
	secret string
}

// NewSigner creates a new Signer from the provided API secret.
func NewSigner(secret string) *Signer {
	return &Signer{secret: secret}
}

// Expiry returns a Unix epoch timestamp in seconds for request expiry. The
// offset controls how many seconds in the future the request expires.
func (s *Signer) Expiry(offsetSeconds int64) int64 {
	if offsetSeconds <= 0 {
		offsetSeconds = defaultExpiry
	}
	return time.Now().Unix() + offsetSeconds
}

// Sign creates an HMAC SHA256 signature for a request. The path must include
// the leading slash, and queryString must not include the leading question
// mark.
func (s *Signer) Sign(method, path, queryString string, expiry int64, body string) string {
	method = strings.ToUpper(method)
	message := fmt.Sprintf("%s%s%d%s", path, queryString, expiry, body)
	mac := hmac.New(sha256.New, []byte(s.secret))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}
