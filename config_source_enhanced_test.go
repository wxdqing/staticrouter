package staticrouter

import (
	"strings"
	"testing"
)

func TestLoadRouteSnapshotWithTypeAndField(t *testing.T) {
	raw := `
<routes version="3" scope="qa">
  <route>
    <kinds>
      <kind>player</kind>
    </kinds>
    <nodes>
      <node node_id="game-node-1" type="game">
        <route_keys field="player_id">
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

	snapshot, err := LoadRouteSnapshot(ConfigModeXML, strings.NewReader(raw))
	if err != nil {
		t.Fatalf("load returned error: %v", err)
	}
	if len(snapshot.GetRoutes()) != 1 {
		t.Fatalf("expected 1 expanded route, got %d", len(snapshot.GetRoutes()))
	}
	rec := snapshot.GetRoutes()[0]
	if rec.GetNodeType() != "game" {
		t.Fatalf("expected node_type game, got %s", rec.GetNodeType())
	}
	if rec.GetRouteKeyField() != "player_id" {
		t.Fatalf("expected route_key_field player_id, got %s", rec.GetRouteKeyField())
	}
}

func TestLoadRouteSnapshotDefaultsTypeWhenMissing(t *testing.T) {
	raw := `{
  "version": 1,
  "scope": "dev",
  "routes": [{
    "kinds": { "kind": ["player"] },
    "nodes": {
      "node": [{
        "node_id": "game-1",
        "route_keys": { "keys": { "key": [1001] } }
      }]
    }
  }]
}`
	snapshot, err := LoadRouteSnapshot(ConfigModeJSON, strings.NewReader(raw))
	if err != nil {
		t.Fatalf("load returned error: %v", err)
	}
	if snapshot.GetRoutes()[0].GetNodeType() != "game" {
		t.Fatalf("expected default node_type game")
	}
}
