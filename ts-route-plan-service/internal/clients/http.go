package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/FudanSELab/train-ticket/ts-route-plan-service/internal/discovery"
)

type transport struct {
	resolver discovery.Resolver
	client   *http.Client
}

func newTransport(resolver discovery.Resolver, client *http.Client) transport {
	if client == nil {
		client = &http.Client{}
	}
	return transport{resolver: resolver, client: client}
}

func (t transport) exchange(ctx context.Context, serviceName, method, path string, input, output any) error {
	baseURL, err := t.resolver.Resolve(ctx, serviceName)
	if err != nil {
		return err
	}
	var body *bytes.Reader
	if input == nil {
		body = bytes.NewReader(nil)
	} else {
		encoded, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(baseURL, "/")+path, body)
	if err != nil {
		return err
	}
	if input != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := t.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("%s %s returned %s", serviceName, path, response.Status)
	}
	if response.Body == nil {
		return fmt.Errorf("%s %s returned no body", serviceName, path)
	}
	return json.NewDecoder(response.Body).Decode(output)
}
