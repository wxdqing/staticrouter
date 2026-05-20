package staticrouter

import (
	"context"
	"testing"
	"time"
)

type stubStore struct {
	getSnapshotFn     func(context.Context, string) (*RouteSnapshot, error)
	replaceSnapshotFn func(context.Context, *RouteSnapshot) error
	watchFn           func(context.Context, string) (<-chan *RouteSnapshot, error)
}

func (s *stubStore) GetSnapshot(ctx context.Context, scope string) (*RouteSnapshot, error) {
	if s.getSnapshotFn != nil {
		return s.getSnapshotFn(ctx, scope)
	}
	return nil, nil
}

func (s *stubStore) ReplaceSnapshot(ctx context.Context, snapshot *RouteSnapshot) error {
	if s.replaceSnapshotFn != nil {
		return s.replaceSnapshotFn(ctx, snapshot)
	}
	return nil
}

func (s *stubStore) Watch(ctx context.Context, scope string) (<-chan *RouteSnapshot, error) {
	if s.watchFn != nil {
		return s.watchFn(ctx, scope)
	}
	ch := make(chan *RouteSnapshot)
	close(ch)
	return ch, nil
}

func TestRouterReplaceAllAndGet(t *testing.T) {
	router := NewRouter(nil)
	if err := router.ReplaceAll(context.Background(), &RouteSnapshot{
		Version:  1,
		Checksum: "manual",
		Routes: []*RouteRecord{
			{
				Kind:      "player",
				NodeType:  "game",
				RouteKeys: []int32{1001, 1002},
				NodeId:    "node-a",
			},
		},
	}); err != nil {
		t.Fatalf("replace all returned error: %v", err)
	}

	got, ok := router.Get(&RouteContext{Kind: "player", NodeType: "game", RouteKey: 1002})
	if !ok || got.GetNodeId() != "node-a" {
		t.Fatalf("expected node-a route")
	}
}

func TestRouterRangeLookupUsesSortedRanges(t *testing.T) {
	router := NewRouter(nil)
	if err := router.ReplaceAll(context.Background(), &RouteSnapshot{
		Version: 1,
		Routes: []*RouteRecord{
			{Kind: "player", NodeType: "game", RouteKeyStart: 3000, RouteKeyEnd: 3999, NodeId: "node-c"},
			{Kind: "player", NodeType: "game", RouteKeyStart: 1000, RouteKeyEnd: 1999, NodeId: "node-a"},
			{Kind: "player", NodeType: "game", RouteKeyStart: 2000, RouteKeyEnd: 2999, NodeId: "node-b"},
		},
	}); err != nil {
		t.Fatalf("replace all returned error: %v", err)
	}

	got, ok := router.Get(&RouteContext{Kind: "player", NodeType: "game", RouteKey: 2500})
	if !ok || got.GetNodeId() != "node-b" {
		t.Fatalf("expected node-b route")
	}
}

func TestRouterRejectsSnapshotRollbackOnWatch(t *testing.T) {
	ch := make(chan *RouteSnapshot, 2)
	store := &stubStore{
		getSnapshotFn: func(context.Context, string) (*RouteSnapshot, error) {
			return &RouteSnapshot{
				Version: 10,
				Scope:   "qa",
				Routes: []*RouteRecord{
					{Kind: "player", NodeType: "game", RouteKeys: []int32{1}, NodeId: "node-v10"},
				},
			}, nil
		},
		watchFn: func(context.Context, string) (<-chan *RouteSnapshot, error) {
			return ch, nil
		},
	}

	router := NewRouterWithConfig(Config{Scope: "qa"}, store)
	if err := router.Start(context.Background()); err != nil {
		t.Fatalf("start returned error: %v", err)
	}

	ch <- &RouteSnapshot{
		Version: 9,
		Scope:   "qa",
		Routes: []*RouteRecord{
			{Kind: "player", NodeType: "game", RouteKeys: []int32{1}, NodeId: "node-v9"},
		},
	}

	got, ok := router.Get(&RouteContext{Kind: "player", NodeType: "game", RouteKey: 1})
	if !ok || got.GetNodeId() != "node-v10" {
		t.Fatalf("expected v10 snapshot to remain active")
	}
}

func TestRouterReloadsLatestSnapshotWhenWatchChannelCloses(t *testing.T) {
	firstWatch := make(chan *RouteSnapshot)
	store := &stubStore{}
	loadCount := 0
	store.getSnapshotFn = func(context.Context, string) (*RouteSnapshot, error) {
		loadCount++
		if loadCount == 1 {
			return &RouteSnapshot{
				Version: 1,
				Scope:   "qa",
				Routes: []*RouteRecord{
					{Kind: "player", NodeType: "game", RouteKeys: []int32{1}, NodeId: "node-v1"},
				},
			}, nil
		}
		return &RouteSnapshot{
			Version: 2,
			Scope:   "qa",
			Routes: []*RouteRecord{
				{Kind: "player", NodeType: "game", RouteKeys: []int32{1}, NodeId: "node-v2"},
			},
		}, nil
	}
	store.watchFn = func(context.Context, string) (<-chan *RouteSnapshot, error) {
		return firstWatch, nil
	}

	router := NewRouterWithConfig(Config{Scope: "qa"}, store)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := router.Start(ctx); err != nil {
		t.Fatalf("start returned error: %v", err)
	}

	close(firstWatch)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		got, ok := router.Get(&RouteContext{Kind: "player", NodeType: "game", RouteKey: 1})
		if ok && got.GetNodeId() == "node-v2" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("expected router to reload latest snapshot after watch close")
}

func TestRouterLoadsSnapshotByScope(t *testing.T) {
	store := &stubStore{
		getSnapshotFn: func(_ context.Context, scope string) (*RouteSnapshot, error) {
			if scope != "prod" {
				t.Fatalf("expected scope prod, got %s", scope)
			}
			return &RouteSnapshot{
				Version: 1,
				Scope:   "prod",
				Routes: []*RouteRecord{
					{Kind: "player", NodeType: "game", RouteKeys: []int32{1}, NodeId: "node-prod"},
				},
			}, nil
		},
		watchFn: func(_ context.Context, scope string) (<-chan *RouteSnapshot, error) {
			ch := make(chan *RouteSnapshot)
			close(ch)
			return ch, nil
		},
	}

	router := NewRouterWithConfig(Config{Scope: "prod"}, store)
	if err := router.Start(context.Background()); err != nil {
		t.Fatalf("start returned error: %v", err)
	}

	got, ok := router.Get(&RouteContext{Kind: "player", NodeType: "game", RouteKey: 1})
	if !ok || got.GetNodeId() != "node-prod" {
		t.Fatalf("expected node-prod route")
	}
}
