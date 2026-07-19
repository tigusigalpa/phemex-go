package phemex

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"
)

func TestSigner_Expiry(t *testing.T) {
	s := NewSigner("secret")
	now := time.Now().Unix()
	got := s.Expiry(60)
	if got <= now {
		t.Fatalf("expected expiry %d to be after now %d", got, now)
	}
	if got > now+120 {
		t.Fatalf("expiry %d is too far in the future", got)
	}
}

func TestSigner_Sign(t *testing.T) {
	s := NewSigner("api-secret")
	got := s.Sign("GET", "/accounts/accountPositions", "currency=BTC", 1893456000, "")

	mac := hmac.New(sha256.New, []byte("api-secret"))
	mac.Write([]byte("/accounts/accountPositionscurrency=BTC1893456000"))
	want := hex.EncodeToString(mac.Sum(nil))

	if got != want {
		t.Fatalf("signature mismatch\n got: %s\nwant: %s", got, want)
	}
}

func TestSigner_SignPostWithBody(t *testing.T) {
	s := NewSigner("api-secret")
	got := s.Sign("POST", "/assets/transfer", "", 1893456000, `{"currency":"BTC","amount":"0.1"}`)

	mac := hmac.New(sha256.New, []byte("api-secret"))
	mac.Write([]byte("/assets/transfer1893456000{\"currency\":\"BTC\",\"amount\":\"0.1\"}"))
	want := hex.EncodeToString(mac.Sum(nil))

	if got != want {
		t.Fatalf("signature mismatch\n got: %s\nwant: %s", got, want)
	}
}

func TestSigner_MethodIsUppercased(t *testing.T) {
	s := NewSigner("secret")
	lower := s.Sign("post", "/path", "", 1, "")
	upper := s.Sign("POST", "/path", "", 1, "")
	if lower != upper {
		t.Fatalf("signature should be case-insensitive for HTTP method")
	}
}
