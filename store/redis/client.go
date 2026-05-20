package redis

import (
	"strings"

	goredis "github.com/redis/go-redis/v9"
)

type Config struct {
	Host      string
	Password  string
	Index     int
	IsCluster bool
}

func NewClient(cfg Config) goredis.UniversalClient {
	if cfg.IsCluster {
		return goredis.NewClusterClient(&goredis.ClusterOptions{
			Addrs:    splitAddrs(cfg.Host),
			Password: cfg.Password,
		})
	}

	return goredis.NewClient(&goredis.Options{
		Addr:     splitAddrs(cfg.Host)[0],
		Password: cfg.Password,
		DB:       cfg.Index,
	})
}

func splitAddrs(host string) []string {
	parts := strings.Split(host, ",")
	addrs := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		addrs = append(addrs, trimmed)
	}
	return addrs
}
