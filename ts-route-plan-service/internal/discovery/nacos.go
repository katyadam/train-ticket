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
	"sync"
	"time"
)

type Resolver interface {
	Resolve(ctx context.Context, serviceName string) (string, error)
}

type Nacos struct {
	addresses []string
	static    map[string]string
	client    *http.Client
	mu        sync.Mutex
	next      map[string]int
	service   string
	ip        string
	port      int
	stopBeat  context.CancelFunc
}

func New(addresses []string, static map[string]string, service, ip string, port int) *Nacos {
	normalized := make([]string, 0, len(addresses))
	for _, address := range addresses {
		normalized = append(normalized, normalizeAddress(address))
	}
	return &Nacos{
		addresses: normalized,
		static:    static,
		client:    &http.Client{Timeout: 5 * time.Second},
		next:      make(map[string]int),
		service:   service,
		ip:        ip,
		port:      port,
	}
}

func (n *Nacos) Resolve(ctx context.Context, serviceName string) (string, error) {
	if target := strings.TrimRight(n.static[serviceName], "/"); target != "" {
		return target, nil
	}
	var lastErr error
	for _, address := range n.addresses {
		query := url.Values{"serviceName": {serviceName}, "healthyOnly": {"true"}}
		var response instanceList
		if err := n.doJSON(ctx, http.MethodGet, address+"/v1/ns/instance/list?"+query.Encode(), nil, &response); err != nil {
			lastErr = err
			continue
		}
		var hosts []instance
		for _, host := range response.Hosts {
			if host.Healthy && host.Enabled && host.IP != "" && host.Port > 0 {
				hosts = append(hosts, host)
			}
		}
		if len(hosts) == 0 {
			lastErr = fmt.Errorf("nacos returned no healthy instance for %s", serviceName)
			continue
		}
		n.mu.Lock()
		index := n.next[serviceName] % len(hosts)
		n.next[serviceName]++
		n.mu.Unlock()
		return "http://" + net.JoinHostPort(hosts[index].IP, strconv.Itoa(hosts[index].Port)), nil
	}
	if lastErr == nil {
		lastErr = errors.New("no Nacos address configured")
	}
	return "", lastErr
}

func (n *Nacos) Register(ctx context.Context) error {
	if n.service == "" || n.ip == "" || n.port == 0 {
		return errors.New("service name, pod IP, and port are required for Nacos registration")
	}
	values := n.instanceValues()
	var lastErr error
	for _, address := range n.addresses {
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, address+"/v1/ns/instance", strings.NewReader(values.Encode()))
		if err != nil {
			return err
		}
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response, err := n.client.Do(request)
		if err != nil {
			lastErr = err
			continue
		}
		body, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr == nil && response.StatusCode >= 200 && response.StatusCode < 300 && strings.TrimSpace(string(body)) == "ok" {
			beatContext, cancel := context.WithCancel(context.Background())
			n.stopBeat = cancel
			go n.heartbeat(beatContext, address)
			return nil
		}
		lastErr = fmt.Errorf("nacos registration returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	if lastErr == nil {
		lastErr = errors.New("no Nacos address configured")
	}
	return lastErr
}

func (n *Nacos) Deregister(ctx context.Context) error {
	if n.stopBeat != nil {
		n.stopBeat()
	}
	values := n.instanceValues()
	var lastErr error
	for _, address := range n.addresses {
		request, err := http.NewRequestWithContext(ctx, http.MethodDelete, address+"/v1/ns/instance?"+values.Encode(), nil)
		if err != nil {
			return err
		}
		response, err := n.client.Do(request)
		if err != nil {
			lastErr = err
			continue
		}
		body, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr == nil && response.StatusCode >= 200 && response.StatusCode < 300 && strings.TrimSpace(string(body)) == "ok" {
			return nil
		}
		lastErr = fmt.Errorf("nacos deregistration returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	return lastErr
}

func (n *Nacos) heartbeat(ctx context.Context, address string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			beat, _ := json.Marshal(map[string]any{
				"ip": n.ip, "port": n.port, "serviceName": n.service,
				"weight": 1, "healthy": true, "enabled": true,
				"ephemeral": true, "cluster": "DEFAULT",
			})
			values := n.instanceValues()
			values.Set("beat", string(beat))
			request, err := http.NewRequestWithContext(ctx, http.MethodPut, address+"/v1/ns/instance/beat", strings.NewReader(values.Encode()))
			if err != nil {
				continue
			}
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response, err := n.client.Do(request)
			if err == nil {
				io.Copy(io.Discard, response.Body)
				response.Body.Close()
			}
		}
	}
}

func (n *Nacos) instanceValues() url.Values {
	return url.Values{
		"serviceName": {n.service}, "ip": {n.ip}, "port": {strconv.Itoa(n.port)},
		"clusterName": {"DEFAULT"}, "ephemeral": {"true"},
	}
}

func (n *Nacos) doJSON(ctx context.Context, method, endpoint string, body io.Reader, output any) error {
	request, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return err
	}
	response, err := n.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		data, _ := io.ReadAll(response.Body)
		return fmt.Errorf("nacos returned %s: %s", response.Status, strings.TrimSpace(string(data)))
	}
	return json.NewDecoder(response.Body).Decode(output)
}

type instanceList struct {
	Hosts []instance `json:"hosts"`
}

type instance struct {
	IP      string `json:"ip"`
	Port    int    `json:"port"`
	Healthy bool   `json:"healthy"`
	Enabled bool   `json:"enabled"`
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
