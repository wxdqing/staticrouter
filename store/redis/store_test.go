package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/wxdqing/plan/server/staticrouter/model"
)

func TestStoreReplaceAndGetSnapshot(t *testing.T) {
	ctx := context.Background()
	srv := miniredis.RunT(t)

	store := NewWithUniversalClient(goredis.NewUniversalClient(&goredis.UniversalOptions{
		Addrs: []string{srv.Addr()},
	}))

	snapshot := &model.RouteSnapshot{
		Version:  3,
		Scope:    "qa",
		Checksum: "snapshot-checksum",
		Routes: []*model.RouteRecord{
			{
				Kind:      "player",
				NodeType:  "game",
				RouteKeys: []int32{1001, 1002},
				NodeId:    "game-node-1",
			},
		},
	}

	if err := store.ReplaceSnapshot(ctx, snapshot); err != nil {
		t.Fatalf("replace snapshot returned error: %v", err)
	}

	got, err := store.GetSnapshot(ctx, "qa")
	if err != nil {
		t.Fatalf("get snapshot returned error: %v", err)
	}
	if got.GetVersion() != 3 {
		t.Fatalf("expected version 3, got %d", got.GetVersion())
	}
	if got.GetScope() != "qa" {
		t.Fatalf("expected scope qa, got %s", got.GetScope())
	}
	if got.GetChecksum() != "snapshot-checksum" {
		t.Fatalf("expected checksum snapshot-checksum, got %s", got.GetChecksum())
	}
	if len(got.GetRoutes()) != 1 {
		t.Fatalf("expected 1 route, got %d", len(got.GetRoutes()))
	}
}

func TestStoreWatchReplaysStreamEvents(t *testing.T) {
	ctx := context.Background()
	srv := miniredis.RunT(t)

	store := NewWithUniversalClient(goredis.NewUniversalClient(&goredis.UniversalOptions{
		Addrs: []string{srv.Addr()},
	}))

	if err := store.ReplaceSnapshot(ctx, &model.RouteSnapshot{
		Version: 1,
		Scope:   "qa",
		Routes: []*model.RouteRecord{
			{Kind: "player", NodeType: "game", RouteKeys: []int32{1}, NodeId: "node-a"},
		},
	}); err != nil {
		t.Fatalf("replace snapshot returned error: %v", err)
	}

	ch, err := store.Watch(ctx, "qa")
	if err != nil {
		t.Fatalf("watch returned error: %v", err)
	}

	if err := store.ReplaceSnapshot(ctx, &model.RouteSnapshot{
		Version: 2,
		Scope:   "qa",
		Routes: []*model.RouteRecord{
			{Kind: "player", NodeType: "game", RouteKeys: []int32{1}, NodeId: "node-b"},
		},
	}); err != nil {
		t.Fatalf("replace snapshot returned error: %v", err)
	}

	select {
	case snapshot := <-ch:
		if snapshot.GetVersion() == 2 {
			return
		}
		select {
		case snapshot = <-ch:
			if snapshot.GetVersion() != 2 {
				t.Fatalf("expected version 2, got %d", snapshot.GetVersion())
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("expected second replayed snapshot event")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("expected replayable stream snapshot event")
	}
}

func TestStoreReplaceSnapshotRejectsVersionRollback(t *testing.T) {
	ctx := context.Background()
	srv := miniredis.RunT(t)

	store := NewWithUniversalClient(goredis.NewUniversalClient(&goredis.UniversalOptions{
		Addrs: []string{srv.Addr()},
	}))

	current := &model.RouteSnapshot{
		Version: 5,
		Scope:   "qa",
		Routes: []*model.RouteRecord{
			{Kind: "player", NodeType: "game", RouteKeys: []int32{1}, NodeId: "node-v5"},
		},
	}
	if err := store.ReplaceSnapshot(ctx, current); err != nil {
		t.Fatalf("replace snapshot returned error: %v", err)
	}

	rollback := &model.RouteSnapshot{
		Version: 4,
		Scope:   "qa",
		Routes: []*model.RouteRecord{
			{Kind: "player", NodeType: "game", RouteKeys: []int32{1}, NodeId: "node-v4"},
		},
	}
	if err := store.ReplaceSnapshot(ctx, rollback); err == nil {
		t.Fatalf("expected rollback to be rejected")
	}

	got, err := store.GetSnapshot(ctx, "qa")
	if err != nil {
		t.Fatalf("get snapshot returned error: %v", err)
	}
	if got.GetVersion() != 5 {
		t.Fatalf("expected version 5 to remain, got %d", got.GetVersion())
	}
}
