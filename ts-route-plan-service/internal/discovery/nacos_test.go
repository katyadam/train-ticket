package discovery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestNacosRegistrationResolutionAndDeregistrationProtocol(t *testing.T) {
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		methods = append(methods, request.Method+" "+request.URL.Path)
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/nacos/v1/ns/instance":
			_ = request.ParseForm()
			if request.Form.Get("serviceName") != "ts-route-plan-service" || request.Form.Get("ip") != "10.0.0.7" || request.Form.Get("port") != "14578" {
				t.Fatalf("registration form = %#v", request.Form)
			}
			_, _ = writer.Write([]byte("ok"))
		case request.Method == http.MethodGet && request.URL.Path == "/nacos/v1/ns/instance/list":
			_ = json.NewEncoder(writer).Encode(map[string]any{"hosts": []map[string]any{{"ip": "10.0.0.8", "port": 12346, "healthy": true, "enabled": true}}})
		case request.Method == http.MethodDelete && request.URL.Path == "/nacos/v1/ns/instance":
			_, _ = writer.Write([]byte("ok"))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	nacos := New([]string{server.URL}, map[string]string{}, "ts-route-plan-service", "10.0.0.7", 14578)
	if err := nacos.Register(context.Background()); err != nil {
		t.Fatal(err)
	}
	target, err := nacos.Resolve(context.Background(), "ts-travel-service")
	if err != nil {
		t.Fatal(err)
	}
	if target != "http://10.0.0.8:12346" {
		t.Fatalf("target = %q", target)
	}
	if err := nacos.Deregister(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := []string{"POST /nacos/v1/ns/instance", "GET /nacos/v1/ns/instance/list", "DELETE /nacos/v1/ns/instance"}
	if !reflect.DeepEqual(methods, want) {
		t.Fatalf("methods = %v, want %v", methods, want)
	}
}
