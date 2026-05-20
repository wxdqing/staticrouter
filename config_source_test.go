package staticrouter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRouteSnapshotFromConfigFileXML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "routes.xml")
	raw := `
<routes version="7" scope="qa">
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

	snapshot, err := LoadRouteSnapshotFromFile(path)
	if err != nil {
		t.Fatalf("load returned error: %v", err)
	}
	if snapshot.GetVersion() != 7 {
		t.Fatalf("expected version 7, got %d", snapshot.GetVersion())
	}
	if snapshot.GetScope() != "qa" {
		t.Fatalf("expected scope qa, got %s", snapshot.GetScope())
	}
	if snapshot.GetChecksum() == "" {
		t.Fatalf("expected checksum to be generated")
	}
	if len(snapshot.GetRoutes()) != 2 {
		t.Fatalf("expected 2 expanded routes, got %d", len(snapshot.GetRoutes()))
	}
}

func TestParseConfigDocumentXML(t *testing.T) {
	raw := `
<routes version="9" scope="prod">
  <route>
    <kinds>
      <kind>player</kind>
    </kinds>
    <nodes>
      <node node_id="game-node-1">
        <route_keys>
          <keys>
            <key>1001</key>
            <key>1002</key>
            <key>1003</key>
          </keys>
        </route_keys>
      </node>
    </nodes>
  </route>
</routes>`

	doc, err := Parse(ConfigModeXML, strings.NewReader(raw))
	if err != nil {
		t.Fatalf("parse returned error: %v", err)
	}
	if doc.Version != 9 {
		t.Fatalf("expected version 9, got %d", doc.Version)
	}
	if doc.Scope != "prod" {
		t.Fatalf("expected scope prod, got %s", doc.Scope)
	}
	if len(doc.Routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(doc.Routes))
	}
	if len(doc.Routes[0].Kinds.Kinds) != 1 {
		t.Fatalf("expected 1 kind, got %d", len(doc.Routes[0].Kinds.Kinds))
	}
}

func TestLoadRouteSnapshotFromJSONFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "routes.json")
	raw := `{
  "version": 8,
  "scope": "dev",
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

	snapshot, err := LoadRouteSnapshotFromFile(path)
	if err != nil {
		t.Fatalf("load returned error: %v", err)
	}
	if snapshot.GetVersion() != 8 {
		t.Fatalf("expected version 8, got %d", snapshot.GetVersion())
	}
	if snapshot.GetScope() != "dev" {
		t.Fatalf("expected scope dev, got %s", snapshot.GetScope())
	}
	if len(snapshot.GetRoutes()) != 2 {
		t.Fatalf("expected 2 expanded routes, got %d", len(snapshot.GetRoutes()))
	}
}
