package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FudanSELab/train-ticket/ts-route-plan-service/internal/service"
)

func TestPublicSurfaceAndOptionalJWTValidation(t *testing.T) {
	router := NewRouter(&service.Service{})

	tests := []struct {
		name          string
		method        string
		path          string
		authorization string
		status        int
		body          string
	}{
		{"welcome", http.MethodGet, basePath + "/welcome", "", http.StatusOK, "Welcome to [ RoutePlan Service ] !"},
		{"non bearer is anonymous", http.MethodGet, basePath + "/welcome", "Basic abc", http.StatusOK, "Welcome to [ RoutePlan Service ] !"},
		{"invalid supplied bearer is rejected", http.MethodGet, basePath + "/welcome", "Bearer invalid", http.StatusUnauthorized, ""},
		{"automatic docs are absent", http.MethodGet, "/openapi.json", "", http.StatusNotFound, "404 page not found\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, nil)
			request.Header.Set("Authorization", test.authorization)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.status || response.Body.String() != test.body {
				t.Fatalf("got status=%d body=%q, want status=%d body=%q", response.Code, response.Body.String(), test.status, test.body)
			}
		})
	}
}

func TestCORSPreflight(t *testing.T) {
	router := NewRouter(&service.Service{})
	request := httptest.NewRequest(http.MethodOptions, basePath+"/routePlan/cheapestRoute", nil)
	request.Header.Set("Origin", "https://example.test")
	request.Header.Set("Access-Control-Request-Method", "POST")
	request.Header.Set("Access-Control-Request-Headers", "authorization,content-type")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("allow origin = %q", response.Header().Get("Access-Control-Allow-Origin"))
	}
	if response.Header().Get("Access-Control-Allow-Methods") != "POST" {
		t.Fatalf("allow methods = %q", response.Header().Get("Access-Control-Allow-Methods"))
	}
	if response.Header().Get("Access-Control-Max-Age") != "3600" {
		t.Fatalf("max age = %q", response.Header().Get("Access-Control-Max-Age"))
	}
}
