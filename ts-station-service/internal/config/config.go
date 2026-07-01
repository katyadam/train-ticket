package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

const (
	ServiceName = "ts-station-service"
	DefaultPort = 12345
)

type Config struct {
	Port         int
	PodIP        string
	NacosEnabled bool
	NacosAddrs   []string
	MySQLDSN     string
}

func Load() Config {
	addresses := os.Getenv("NACOS_ADDRS")
	if addresses == "" {
		addresses = "nacos-0.nacos-headless.default.svc.cluster.local,nacos-1.nacos-headless.default.svc.cluster.local,nacos-2.nacos-headless.default.svc.cluster.local"
	}
	port := envInt("PORT", DefaultPort)
	return Config{
		Port:         port,
		PodIP:        os.Getenv("POD_IP"),
		NacosEnabled: strings.ToLower(os.Getenv("NACOS_ENABLED")) != "false",
		NacosAddrs:   split(addresses),
		MySQLDSN:     mysqlDSN(),
	}
}

func mysqlDSN() string {
	if configured := os.Getenv("STATION_MYSQL_DSN"); configured != "" {
		return configured
	}
	configuration := mysql.NewConfig()
	configuration.User = env("STATION_MYSQL_USER", "root")
	configuration.Passwd = env("STATION_MYSQL_PASSWORD", "Abcd1234#")
	configuration.Net = "tcp"
	configuration.Addr = netAddress(env("STATION_MYSQL_HOST", "ts-station-mysql"), env("STATION_MYSQL_PORT", "3306"))
	configuration.DBName = env("STATION_MYSQL_DATABASE", "ts-station-mysql")
	configuration.Params = map[string]string{"charset": "utf8mb4"}
	configuration.Loc = fixedShanghai()
	return configuration.FormatDSN()
}

func fixedShanghai() *time.Location { return time.FixedZone("Asia/Shanghai", 8*60*60) }

func netAddress(host, port string) string {
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		return "[" + host + "]:" + port
	}
	return host + ":" + port
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
func envInt(name string, fallback int) int {
	if value, err := strconv.Atoi(os.Getenv(name)); err == nil && value > 0 {
		return value
	}
	return fallback
}
func split(value string) []string {
	var result []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}
