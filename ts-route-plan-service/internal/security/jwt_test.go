package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"
)

func TestValidateBearer(t *testing.T) {
	now := time.Unix(2_000_000_000, 0)
	valid := signedToken(`{"alg":"HS256","typ":"JWT"}`, `{"sub":"user","roles":["ROLE_USER"],"exp":2000000060}`, "secret")
	invalid := signedToken(`{"alg":"HS256","typ":"JWT"}`, `{"sub":"user","roles":["ROLE_USER"],"exp":2000000060}`, "wrong")
	expired := signedToken(`{"alg":"HS256","typ":"JWT"}`, `{"sub":"user","roles":["ROLE_USER"],"exp":1999999999}`, "secret")

	for name, test := range map[string]struct {
		header string
		valid  bool
	}{
		"none":              {"", true},
		"non bearer":        {"Basic abc", true},
		"valid":             {"Bearer " + valid, true},
		"invalid signature": {"Bearer " + invalid, false},
		"expired":           {"Bearer " + expired, false},
		"malformed":         {"Bearer nonsense", false},
	} {
		t.Run(name, func(t *testing.T) {
			err := ValidateBearer(test.header, now)
			if (err == nil) != test.valid {
				t.Fatalf("valid=%v, error=%v", test.valid, err)
			}
		})
	}
}

func signedToken(header, payload, key string) string {
	headerPart := base64.RawURLEncoding.EncodeToString([]byte(header))
	payloadPart := base64.RawURLEncoding.EncodeToString([]byte(payload))
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(headerPart + "." + payloadPart))
	return headerPart + "." + payloadPart + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
