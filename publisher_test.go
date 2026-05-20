package staticrouter

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type publisherStubStore struct {
	replaced *RouteSnapshot
}

func (s *publisherStubStore) GetSnapshot(ctx context.Context, scope string) (*RouteSnapshot, error) {
	return nil, nil
}

func (s *publisherStubStore) ReplaceSnapshot(ctx context.Context, snapshot *RouteSnapshot) error {
	s.replaced = snapshot
	return nil
}

func (s *publisherStubStore) Watch(ctx context.Context, scope string) (<-chan *RouteSnapshot, error) {
	ch := make(chan *RouteSnapshot)
	close(ch)
	return ch, nil
}

func TestPublisherPublishXMLFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "routes.xml")
	raw := `
<routes version="11" scope="qa">
  <route>
    <kinds>
      <kind>player</kind>
      <kind>mail</kind>
    </kinds>
    <nodes>
      <node node_id="game-node-1">
        <route_keys>
          <keys>
            <key>1001</key>
            <key>1002</key>
          </keys>
          <ranges>
            <range start="2000" end="2099" />
          </ranges>
        </route_keys>
      </node>
    </nodes>
  </route>
</routes>`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("write file returned error: %v", err)
	}

	store := &publisherStubStore{}
	publisher := NewPublisher(store)
	if err := publisher.PublishFile(context.Background(), path); err != nil {
		t.Fatalf("publish returned error: %v", err)
	}

	if store.replaced == nil {
		t.Fatalf("expected snapshot to be published")
	}
	if store.replaced.GetVersion() != 11 {
		t.Fatalf("expected version 11, got %d", store.replaced.GetVersion())
	}
	if store.replaced.GetScope() != "qa" {
		t.Fatalf("expected scope qa, got %s", store.replaced.GetScope())
	}
	if store.replaced.GetChecksum() == "" {
		t.Fatalf("expected checksum to be present")
	}
	if len(store.replaced.GetRoutes()) != 2 {
		t.Fatalf("expected 2 expanded routes, got %d", len(store.replaced.GetRoutes()))
	}
}

func TestPublisherPublishConfigFileJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "routes.json")
	raw := `{
  "version": 12,
  "scope": "review",
  "routes": [
    {
      "kinds": {
        "kind": ["player", "mail"]
      },
      "nodes": {
        "node": [
          {
            "node_id": "game-node-1",
            "route_keys": {
              "keys": {
                "key": [1001, 1002]
              },
              "ranges": {
                "range": [
                  { "start": 2000, "end": 2099 }
                ]
              }
            }
          }
        ]
      }
    }
  ]
}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("write file returned error: %v", err)
	}

	store := &publisherStubStore{}
	publisher := NewPublisher(store)
	if err := publisher.PublishFile(context.Background(), path); err != nil {
		t.Fatalf("publish returned error: %v", err)
	}
	if store.replaced == nil {
		t.Fatalf("expected snapshot to be published")
	}
	if store.replaced.GetScope() != "review" {
		t.Fatalf("expected scope review, got %s", store.replaced.GetScope())
	}
}
