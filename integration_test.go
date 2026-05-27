package staticrouter_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/wxdqing/staticrouter"
	redisstore "github.com/wxdqing/staticrouter/store/redis"
)

func TestBDDGivenSnapshotInRedisWhenRouterStartsThenLookupUsesLocalMemory(t *testing.T) {
	store := newIntegrationStore(t)
	if store == nil {
		t.Skip("integration redis not configured")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	current, err := store.GetSnapshot(ctx, "qa")
	if err != nil {
		t.Fatalf("get snapshot returned error: %v", err)
	}
	baseVersion := int64(1)
	if current != nil && current.GetVersion() >= baseVersion {
		baseVersion = current.GetVersion() + 1
	}

	snapshot := &staticrouter.RouteSnapshot{
		Version: baseVersion,
		Scope:   "qa",
		Routes: []*staticrouter.RouteRecord{
			{Kind: "player", NodeType: "game", RouteKeys: []int32{1001, 1002}, NodeId: "node-a"},
			{Kind: "mail", NodeType: "game", RouteKeyStart: 2000, RouteKeyEnd: 2099, NodeId: "node-b"},
		},
	}
	snapshot, err = staticrouter.NormalizeSnapshot(snapshot)
	if err != nil {
		t.Fatalf("normalize returned error: %v", err)
	}
	if err := store.ReplaceSnapshot(ctx, snapshot); err != nil {
		t.Fatalf("replace snapshot returned error: %v", err)
	}

	router := staticrouter.NewRouterWithConfig(staticrouter.Config{Scope: "qa"}, store)
	if err := router.Start(ctx); err != nil {
		t.Fatalf("start returned error: %v", err)
	}

	got, ok := router.Get(&staticrouter.RouteContext{Kind: "player", NodeType: "game", RouteKey: 1002})
	if !ok || got.GetNodeId() != "node-a" {
		t.Fatalf("expected node-a exact route")
	}

	got, ok = router.Get(&staticrouter.RouteContext{Kind: "mail", NodeType: "game", RouteKey: 2005})
	if !ok || got.GetNodeId() != "node-b" {
		t.Fatalf("expected node-b range route")
	}
}

func TestBDDGivenNewSnapshotWhenWatchReceivesStreamThenRouterAtomicallySwaps(t *testing.T) {
	store := newIntegrationStore(t)
	if store == nil {
		t.Skip("integration redis not configured")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	current, err := store.GetSnapshot(ctx, "qa")
	if err != nil {
		t.Fatalf("get snapshot returned error: %v", err)
	}
	baseVersion := int64(1)
	if current != nil && current.GetVersion() >= baseVersion {
		baseVersion = current.GetVersion() + 1
	}

	initial := &staticrouter.RouteSnapshot{
		Version: baseVersion,
		Scope:   "qa",
		Routes: []*staticrouter.RouteRecord{
			{Kind: "player", NodeType: "game", RouteKeys: []int32{3001}, NodeId: "node-v1"},
		},
	}
	initial, err = staticrouter.NormalizeSnapshot(initial)
	if err != nil {
		t.Fatalf("normalize returned error: %v", err)
	}
	if err := store.ReplaceSnapshot(ctx, initial); err != nil {
		t.Fatalf("replace snapshot returned error: %v", err)
	}

	router := staticrouter.NewRouterWithConfig(staticrouter.Config{Scope: "qa"}, store)
	if err := router.Start(ctx); err != nil {
		t.Fatalf("start returned error: %v", err)
	}

	next := &staticrouter.RouteSnapshot{
		Version: baseVersion + 1,
		Scope:   "qa",
		Routes: []*staticrouter.RouteRecord{
			{Kind: "player", NodeType: "game", RouteKeys: []int32{3001}, NodeId: "node-v2"},
		},
	}
	next, err = staticrouter.NormalizeSnapshot(next)
	if err != nil {
		t.Fatalf("normalize returned error: %v", err)
	}
	if err := store.ReplaceSnapshot(ctx, next); err != nil {
		t.Fatalf("replace snapshot returned error: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		got, ok := router.Get(&staticrouter.RouteContext{Kind: "player", NodeType: "game", RouteKey: 3001})
		if ok && got.GetNodeId() == "node-v2" {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("expected router to atomically swap to node-v2")
}

func TestBDDGivenHigherVersionInRedisWhenLowerVersionPublishesThenSnapshotDoesNotRollback(t *testing.T) {
	store := newIntegrationStore(t)
	if store == nil {
		t.Skip("integration redis not configured")
	}

	ctx := context.Background()

	current, err := store.GetSnapshot(ctx, "qa")
	if err != nil {
		t.Fatalf("get snapshot returned error: %v", err)
	}
	baseVersion := int64(10)
	if current != nil && current.GetVersion() >= baseVersion {
		baseVersion = current.GetVersion() + 10
	}

	high := &staticrouter.RouteSnapshot{
		Version: baseVersion,
		Scope:   "qa",
		Routes: []*staticrouter.RouteRecord{
			{Kind: "player", NodeType: "game", RouteKeys: []int32{9001}, NodeId: "node-high"},
		},
	}
	high, err = staticrouter.NormalizeSnapshot(high)
	if err != nil {
		t.Fatalf("normalize returned error: %v", err)
	}
	if err := store.ReplaceSnapshot(ctx, high); err != nil {
		t.Fatalf("replace high snapshot returned error: %v", err)
	}

	low := &staticrouter.RouteSnapshot{
		Version: baseVersion - 1,
		Scope:   "qa",
		Routes: []*staticrouter.RouteRecord{
			{Kind: "player", NodeType: "game", RouteKeys: []int32{9001}, NodeId: "node-low"},
		},
	}
	low, err = staticrouter.NormalizeSnapshot(low)
	if err != nil {
		t.Fatalf("normalize returned error: %v", err)
	}
	if err := store.ReplaceSnapshot(ctx, low); err == nil {
		t.Fatalf("expected rollback publish to fail")
	}

	got, err := store.GetSnapshot(ctx, "qa")
	if err != nil {
		t.Fatalf("get snapshot returned error: %v", err)
	}
	if got.GetVersion() != baseVersion {
		t.Fatalf("expected version %d to remain, got %d", baseVersion, got.GetVersion())
	}
	if got.GetRoutes()[0].GetNodeId() != "node-high" {
		t.Fatalf("expected node-high to remain, got %s", got.GetRoutes()[0].GetNodeId())
	}
}

func newIntegrationStore(t *testing.T) *redisstore.Store {
	t.Helper()

	if os.Getenv("STATICROUTER_RUN_INTEGRATION") == "" {
		return nil
	}

	host := os.Getenv("STATICROUTER_REDIS_HOST")
	if host == "" {
		host = "192.168.0.138:7000"
	}

	password := os.Getenv("STATICROUTER_REDIS_PASSWORD")
	if password == "" {
		password = "123456"
	}

	return redisstore.New(redisstore.Config{
		Host:      host,
		Password:  password,
		Index:     0,
		IsCluster: true,
	})
}
