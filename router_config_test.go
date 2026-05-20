package staticrouter

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRouterReplaceAllFromConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "routes.xml")
	raw := `
<routes version="2" scope="qa">
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

	router := NewRouterWithConfig(Config{Scope: "qa"}, nil)
	if err := router.ReplaceAllFromFile(context.Background(), path); err != nil {
		t.Fatalf("replace from file returned error: %v", err)
	}

	got, ok := router.Get(&RouteContext{
		Kind:     "player",
		NodeType: "game",
		RouteKey: 1002,
	})
	if !ok {
		t.Fatalf("expected route loaded from file")
	}
	if got.GetNodeId() != "game-node-1" {
		t.Fatalf("expected game-node-1, got %s", got.GetNodeId())
	}
}
