package redis

import (
	"context"
	"net"
	"testing"
	"time"

	"gitee.com/wxdqing/staticrouter/model"
	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

type recordEvalKeysHook struct {
	keys []string
}

func (h *recordEvalKeysHook) DialHook(next goredis.DialHook) goredis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return next(ctx, network, addr)
	}
}

func (h *recordEvalKeysHook) ProcessHook(next goredis.ProcessHook) goredis.ProcessHook {
	return func(ctx context.Context, cmd goredis.Cmder) error {
		args := cmd.Args()
		if cmd.Name() == "eval" && len(args) >= 3 {
			keyCount, ok := args[2].(int)
			if ok {
				h.keys = h.keys[:0]
				for i := 0; i < keyCount; i++ {
					key, ok := args[3+i].(string)
					if ok {
						h.keys = append(h.keys, key)
					}
				}
			}
		}
		return next(ctx, cmd)
	}
}

func (h *recordEvalKeysHook) ProcessPipelineHook(next goredis.ProcessPipelineHook) goredis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []goredis.Cmder) error {
		return next(ctx, cmds)
	}
}

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

func TestStoreReplaceSnapshotPassesAllScriptKeysExplicitly(t *testing.T) {
	ctx := context.Background()
	srv := miniredis.RunT(t)
	hook := &recordEvalKeysHook{}
	client := goredis.NewUniversalClient(&goredis.UniversalOptions{
		Addrs: []string{srv.Addr()},
	})
	client.AddHook(hook)
	store := NewWithUniversalClient(client)

	if err := store.ReplaceSnapshot(ctx, &model.RouteSnapshot{
		Version: 1,
		Scope:   "qa",
		Routes: []*model.RouteRecord{
			{Kind: "player", NodeType: "game", RouteKeys: []int32{1}, NodeId: "node-a"},
		},
	}); err != nil {
		t.Fatalf("replace snapshot returned error: %v", err)
	}

	want := []string{
		"staticrouter:snapshot:{qa}",
		"staticrouter:events:{qa}",
		"staticrouter:snapshot:{qa}:meta",
	}
	if len(hook.keys) != len(want) {
		t.Fatalf("expected %d script keys, got %d: %v", len(want), len(hook.keys), hook.keys)
	}
	for i := range want {
		if hook.keys[i] != want[i] {
			t.Fatalf("expected script key %d to be %s, got %s", i, want[i], hook.keys[i])
		}
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

func TestStoreWatchersDoNotShareCursor(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := miniredis.RunT(t)

	store := NewWithUniversalClient(goredis.NewUniversalClient(&goredis.UniversalOptions{
		Addrs: []string{srv.Addr()},
	}))

	ch1, err := store.Watch(ctx, "qa")
	if err != nil {
		t.Fatalf("first watch returned error: %v", err)
	}

	if err := store.ReplaceSnapshot(ctx, &model.RouteSnapshot{
		Version: 1,
		Scope:   "qa",
		Routes: []*model.RouteRecord{
			{Kind: "player", NodeType: "game", RouteKeys: []int32{1}, NodeId: "node-a"},
		},
	}); err != nil {
		t.Fatalf("replace snapshot returned error: %v", err)
	}

	expectSnapshotVersion(t, ch1, 1)

	ch2, err := store.Watch(ctx, "qa")
	if err != nil {
		t.Fatalf("second watch returned error: %v", err)
	}
	expectSnapshotVersion(t, ch2, 1)
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

func expectSnapshotVersion(t *testing.T, ch <-chan *model.RouteSnapshot, version int64) {
	t.Helper()

	deadline := time.After(2 * time.Second)
	for {
		select {
		case snapshot, ok := <-ch:
			if !ok {
				t.Fatalf("watch channel closed before version %d", version)
			}
			if snapshot.GetVersion() == version {
				return
			}
		case <-deadline:
			t.Fatalf("expected snapshot version %d", version)
		}
	}
}
