package staticrouter_test

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"staticrouter"
	redisstore "staticrouter/store/redis"
)

func TestUpdateConfigPublishesAndRefreshesDefaultRouter(t *testing.T) {
	xmlContent := `
<routes version="20" scope="qa">
  <route>
    <kinds>
      <kind>player</kind>
    </kinds>
    <nodes>
      <node node_id="game-node-1">
        <route_keys>
          <keys>
            <key>1001</key>
          </keys>
        </route_keys>
      </node>
    </nodes>
  </route>
</routes>`

	if err := staticrouter.Init(
		staticrouter.WithScope("qa"),
	); err != nil {
		t.Fatalf("init returned error: %v", err)
	}
	if _, ok := staticrouter.GetRoute(&staticrouter.RouteContext{Kind: "player", NodeType: "game", RouteKey: 1001}); ok {
		t.Fatalf("expected empty router before update")
	}

	if err := staticrouter.UpdateConfig(staticrouter.ConfigModeXML, []byte(xmlContent)); err != nil {
		t.Fatalf("update config returned error: %v", err)
	}

	got, ok := staticrouter.GetRoute(&staticrouter.RouteContext{Kind: "player", NodeType: "game", RouteKey: 1001})
	if !ok || got.GetNodeId() != "game-node-1" {
		t.Fatalf("expected game-node-1 route")
	}
}

func TestInitWithRedisConfig(t *testing.T) {
	srv := miniredis.RunT(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	xmlContent := `
<routes version="21" scope="qa">
  <route>
    <kinds>
      <kind>player</kind>
    </kinds>
    <nodes>
      <node node_id="game-node-1">
        <route_keys>
          <keys>
            <key>1001</key>
          </keys>
        </route_keys>
      </node>
    </nodes>
  </route>
</routes>`

	if err := staticrouter.Init(
		staticrouter.WithContext(ctx),
		staticrouter.WithScope("qa"),
		staticrouter.WithRedisConfig(staticrouter.RedisConfig{Host: srv.Addr(), Password: "", Index: 0, IsCluster: false}),
	); err != nil {
		t.Fatalf("init returned error: %v", err)
	}
	if err := staticrouter.UpdateConfig(staticrouter.ConfigModeXML, []byte(xmlContent)); err != nil {
		t.Fatalf("update config returned error: %v", err)
	}

	got, ok := staticrouter.GetRoute(&staticrouter.RouteContext{Kind: "player", NodeType: "game", RouteKey: 1001})
	if !ok || got.GetNodeId() != "game-node-1" {
		t.Fatalf("expected game-node-1 route")
	}
}

func TestInitWithStore(t *testing.T) {
	srv := miniredis.RunT(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := goredis.NewClient(&goredis.Options{Addr: srv.Addr()})
	defer client.Close()
	store := redisstore.NewWithUniversalClient(client)

	jsonContent := `{
  "version": 22,
  "scope": "qa",
  "routes": [
    {
      "kinds": {
        "kind": ["player"]
      },
      "nodes": {
        "node": [
          {
            "node_id": "game-node-2",
            "route_keys": {
              "keys": {
                "key": [2001]
              }
            }
          }
        ]
      }
    }
  ]
}`

	if err := staticrouter.Init(
		staticrouter.WithContext(ctx),
		staticrouter.WithScope("qa"),
		staticrouter.WithStore(store),
	); err != nil {
		t.Fatalf("init returned error: %v", err)
	}
	if err := staticrouter.UpdateConfig(staticrouter.ConfigModeJSON, []byte(jsonContent)); err != nil {
		t.Fatalf("update config returned error: %v", err)
	}

	got, ok := staticrouter.GetRoute(&staticrouter.RouteContext{Kind: "player", NodeType: "game", RouteKey: 2001})
	if !ok || got.GetNodeId() != "game-node-2" {
		t.Fatalf("expected game-node-2 route")
	}
}

func TestInitWithRedisInstance(t *testing.T) {
	srv := miniredis.RunT(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := goredis.NewClient(&goredis.Options{Addr: srv.Addr()})
	defer client.Close()

	jsonContent := `{
  "version": 23,
  "scope": "qa",
  "routes": [
    {
      "kinds": {
        "kind": ["player"]
      },
      "nodes": {
        "node": [
          {
            "node_id": "game-node-3",
            "route_keys": {
              "keys": {
                "key": [3001]
              }
            }
          }
        ]
      }
    }
  ]
}`

	if err := staticrouter.Init(
		staticrouter.WithContext(ctx),
		staticrouter.WithScope("qa"),
		staticrouter.WithRedisInstance(client),
	); err != nil {
		t.Fatalf("init returned error: %v", err)
	}
	if err := staticrouter.UpdateConfig(staticrouter.ConfigModeJSON, []byte(jsonContent)); err != nil {
		t.Fatalf("update config returned error: %v", err)
	}

	got, ok := staticrouter.GetRoute(&staticrouter.RouteContext{Kind: "player", NodeType: "game", RouteKey: 3001})
	if !ok || got.GetNodeId() != "game-node-3" {
		t.Fatalf("expected game-node-3 route")
	}
}
