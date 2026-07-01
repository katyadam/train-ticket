package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/FudanSELab/train-ticket/ts-station-service/internal/service"
	"github.com/FudanSELab/train-ticket/ts-station-service/internal/testsupport"
)

func TestAuthorizationRulesIncludingPublicDeleteQuirk(t *testing.T) {
	router := NewRouter(service.New(testsupport.NewRepository()))
	admin := token(`["ROLE_ADMIN"]`)
	user := token(`["ROLE_USER"]`)
	tests := []struct {
		name, method, path, body, authorization string
		status                                  int
	}{
		{"public read", "GET", basePath + "/stations", "", "", 200},
		{"invalid bearer on public read", "GET", basePath + "/stations", "", "Bearer bad", 401},
		{"post without token", "POST", basePath + "/stations", `{"name":"new"}`, "", 403},
		{"post with user", "POST", basePath + "/stations", `{"name":"new"}`, "Bearer " + user, 403},
		{"post with admin", "POST", basePath + "/stations", `{"name":"new"}`, "Bearer " + admin, 201},
		{"delete with ID remains public", "DELETE", basePath + "/stations/missing", "", "", 200},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			request.Header.Set("Authorization", test.authorization)
			if test.body != "" {
				request.Header.Set("Content-Type", "application/json")
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status=%d body=%q, want %d", response.Code, response.Body.String(), test.status)
			}
		})
	}
}

func TestWireSurfaceAndMalformedBodies(t *testing.T) {
	router := NewRouter(service.New(testsupport.NewRepository()))
	welcome := httptest.NewRecorder()
	router.ServeHTTP(welcome, httptest.NewRequest("GET", basePath+"/welcome", nil))
	if welcome.Body.String() != "Welcome to [ Station Service ] !" {
		t.Fatalf("welcome = %q", welcome.Body.String())
	}
	docs := httptest.NewRecorder()
	router.ServeHTTP(docs, httptest.NewRequest("GET", "/openapi.json", nil))
	if docs.Code != 404 {
		t.Fatalf("docs status = %d", docs.Code)
	}
	bad := httptest.NewRecorder()
	request := httptest.NewRequest("POST", basePath+"/stations", bytes.NewBufferString(`{"name":null}`))
	request.Header.Set("Authorization", "Bearer "+token(`["ROLE_ADMIN"]`))
	router.ServeHTTP(bad, request)
	if bad.Code != 400 {
		t.Fatalf("null-name status = %d", bad.Code)
	}
}

func token(roles string) string {
	header := encode(`{"alg":"HS256","typ":"JWT"}`)
	payload := encode(`{"sub":"user","roles":` + roles + `,"exp":` + fmt.Sprint(time.Now().Add(time.Minute).Unix()) + `}`)
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte(header + "." + payload))
	return header + "." + payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func encode(value string) string { return base64.RawURLEncoding.EncodeToString([]byte(value)) }
