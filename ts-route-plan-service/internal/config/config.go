package config

import (
	"os"
	"strconv"
	"strings"
)

const (
	ServiceName = "ts-route-plan-service"
	DefaultPort = 14578
)

type Config struct {
	Port          int
	PodIP         string
	NacosEnabled  bool
	NacosAddrs    []string
	StaticTargets map[string]string
}

func Load() Config {
	port := envInt("PORT", DefaultPort)
	nacosEnabled := strings.ToLower(os.Getenv("NACOS_ENABLED")) != "false"
	addresses := os.Getenv("NACOS_ADDRS")
	if addresses == "" {
		addresses = "nacos-0.nacos-headless.default.svc.cluster.local,nacos-1.nacos-headless.default.svc.cluster.local,nacos-2.nacos-headless.default.svc.cluster.local"
	}
	return Config{
		Port:         port,
		PodIP:        os.Getenv("POD_IP"),
		NacosEnabled: nacosEnabled,
		NacosAddrs:   splitNonEmpty(addresses),
		StaticTargets: map[string]string{
			"ts-travel-service":  strings.TrimRight(os.Getenv("TRAVEL_SERVICE_URL"), "/"),
			"ts-travel2-service": strings.TrimRight(os.Getenv("TRAVEL2_SERVICE_URL"), "/"),
			"ts-route-service":   strings.TrimRight(os.Getenv("ROUTE_SERVICE_URL"), "/"),
		},
	}
}

func envInt(name string, fallback int) int {
	if value, err := strconv.Atoi(os.Getenv(name)); err == nil && value > 0 {
		return value
	}
	return fallback
}

func splitNonEmpty(value string) []string {
	var result []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}
