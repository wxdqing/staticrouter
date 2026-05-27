package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/wxdqing/staticrouter"
	redisstore "github.com/wxdqing/staticrouter/store/redis"
)

func main() {
	var (
		configPath = flag.String("config", "", "path to staticrouter config file (.xml or .json)")
		redisHost  = flag.String("redis", "", "redis host or cluster seed address, e.g. 127.0.0.1:6379")
		password   = flag.String("password", "", "redis password")
		cluster    = flag.Bool("cluster", false, "use redis cluster client")
		db         = flag.Int("db", 0, "redis db index, ignored in cluster mode")
		timeout    = flag.Duration("timeout", 10*time.Second, "publish timeout")
	)

	flag.Parse()

	if *configPath == "" {
		exitf("missing required flag: -config")
	}
	if *redisHost == "" {
		exitf("missing required flag: -redis")
	}

	store := redisstore.New(redisstore.Config{
		Host:      *redisHost,
		Password:  *password,
		Index:     *db,
		IsCluster: *cluster,
	})

	publisher := staticrouter.NewPublisher(store)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	if err := publisher.PublishFile(ctx, *configPath); err != nil {
		exitf("publish failed: %v", err)
	}

	fmt.Printf("published staticrouter config: %s\n", *configPath)
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
