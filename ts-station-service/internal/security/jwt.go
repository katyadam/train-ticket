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

type Authentication struct {
	Present bool
	Roles   map[string]bool
}

func ParseOptionalBearer(authorization string, now time.Time) (Authentication, error) {
	if !strings.HasPrefix(authorization, "Bearer ") {
		return Authentication{}, nil
	}
	parts := strings.Split(strings.TrimPrefix(authorization, "Bearer "), ".")
	if len(parts) != 3 {
		return Authentication{}, errors.New("malformed JWT")
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Authentication{}, err
	}
	var header struct {
		Algorithm string `json:"alg"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil || header.Algorithm != "HS256" {
		return Authentication{}, errors.New("unsupported JWT")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Authentication{}, err
	}
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte(parts[0] + "." + parts[1]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return Authentication{}, errors.New("invalid JWT signature")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Authentication{}, err
	}
	var payload struct {
		Exp   json.Number `json:"exp"`
		Roles []string    `json:"roles"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(payloadBytes)))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return Authentication{}, err
	}
	if payload.Exp == "" {
		return Authentication{}, errors.New("JWT has no expiration")
	}
	expiration, err := payload.Exp.Int64()
	if err != nil {
		return Authentication{}, err
	}
	if time.Unix(expiration, 0).Before(now) {
		return Authentication{}, errors.New("JWT expired")
	}
	roles := make(map[string]bool, len(payload.Roles))
	for _, role := range payload.Roles {
		roles[role] = true
	}
	return Authentication{Present: true, Roles: roles}, nil
}
