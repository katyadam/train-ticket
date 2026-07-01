package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// ValidateBearer reproduces the local HS256 validation used by ts-common JWTFilter.
// A non-Bearer Authorization value is treated exactly like no token.
func ValidateBearer(authorization string, now time.Time) error {
	if !strings.HasPrefix(authorization, "Bearer ") {
		return nil
	}
	token := strings.TrimPrefix(authorization, "Bearer ")
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return errors.New("malformed JWT")
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return err
	}
	var header struct {
		Algorithm string `json:"alg"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil || header.Algorithm != "HS256" {
		return errors.New("unsupported JWT")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return err
	}
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte(parts[0] + "." + parts[1]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return errors.New("invalid JWT signature")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return err
	}
	var payload struct {
		Exp json.Number `json:"exp"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(payloadBytes)))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return err
	}
	if payload.Exp == "" {
		return errors.New("JWT has no expiration")
	}
	exp, err := payload.Exp.Int64()
	if err != nil {
		return err
	}
	if time.Unix(exp, 0).Before(now) {
		return errors.New("JWT expired")
	}
	return nil
}
