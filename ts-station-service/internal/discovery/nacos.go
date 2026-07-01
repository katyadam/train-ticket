package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Registrar struct {
	addresses   []string
	client      *http.Client
	service, ip string
	port        int
	registered  string
	stop        context.CancelFunc
}

func NewRegistrar(addresses []string, service, ip string, port int) *Registrar {
	normalized := make([]string, 0, len(addresses))
	for _, address := range addresses {
		normalized = append(normalized, normalizeAddress(address))
	}
	return &Registrar{addresses: normalized, client: &http.Client{Timeout: 5 * time.Second}, service: service, ip: ip, port: port}
}

func (r *Registrar) Register(ctx context.Context) error {
	if r.ip == "" {
		return errors.New("pod IP is required for Nacos registration")
	}
	var lastErr error
	for _, address := range r.addresses {
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, address+"/v1/ns/instance", strings.NewReader(r.values().Encode()))
		if err != nil {
			return err
		}
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response, err := r.client.Do(request)
		if err != nil {
			lastErr = err
			continue
		}
		body, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr == nil && response.StatusCode >= 200 && response.StatusCode < 300 && strings.TrimSpace(string(body)) == "ok" {
			r.registered = address
			beatContext, cancel := context.WithCancel(context.Background())
			r.stop = cancel
			go r.heartbeat(beatContext)
			return nil
		}
		lastErr = fmt.Errorf("Nacos registration returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	if lastErr == nil {
		lastErr = errors.New("no Nacos address configured")
	}
	return lastErr
}

func (r *Registrar) Deregister(ctx context.Context) error {
	if r.stop != nil {
		r.stop()
	}
	if r.registered == "" {
		return nil
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, r.registered+"/v1/ns/instance?"+r.values().Encode(), nil)
	if err != nil {
		return err
	}
	response, err := r.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Nacos deregistration returned %s", response.Status)
	}
	return nil
}

func (r *Registrar) heartbeat(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			beat, _ := json.Marshal(map[string]any{"ip": r.ip, "port": r.port, "serviceName": r.service, "weight": 1, "healthy": true, "enabled": true, "ephemeral": true, "cluster": "DEFAULT"})
			values := r.values()
			values.Set("beat", string(beat))
			request, err := http.NewRequestWithContext(ctx, http.MethodPut, r.registered+"/v1/ns/instance/beat", strings.NewReader(values.Encode()))
			if err != nil {
				continue
			}
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response, err := r.client.Do(request)
			if err == nil {
				io.Copy(io.Discard, response.Body)
				response.Body.Close()
			}
		}
	}
}

func (r *Registrar) values() url.Values {
	return url.Values{"serviceName": {r.service}, "ip": {r.ip}, "port": {strconv.Itoa(r.port)}, "clusterName": {"DEFAULT"}, "ephemeral": {"true"}}
}

func DetectPodIP(configured string) (string, error) {
	if configured != "" {
		return configured, nil
	}
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, address := range addresses {
		var ip net.IP
		switch value := address.(type) {
		case *net.IPNet:
			ip = value.IP
		case *net.IPAddr:
			ip = value.IP
		}
		if ip != nil && !ip.IsLoopback() && ip.To4() != nil {
			return ip.String(), nil
		}
	}
	return "", errors.New("no non-loopback IPv4 address found")
}

func normalizeAddress(address string) string {
	address = strings.TrimRight(strings.TrimSpace(address), "/")
	if !strings.Contains(address, "://") {
		address = "http://" + address
	}
	parsed, err := url.Parse(address)
	if err == nil && parsed.Port() == "" {
		parsed.Host = net.JoinHostPort(parsed.Hostname(), "8848")
		address = strings.TrimRight(parsed.String(), "/")
	}
	if !strings.HasSuffix(address, "/nacos") {
		address += "/nacos"
	}
	return address
}
