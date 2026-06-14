package validate

import (
	"strings"
	"testing"

	"github.com/wxdqing/staticrouter/source"
)

type stubRegistry struct {
	nodeTypes map[string]struct{}
	kinds     map[string]string
	fields    map[string]struct{}
}

func (r stubRegistry) HasNodeType(code string) bool {
	_, ok := r.nodeTypes[code]
	return ok
}

func (r stubRegistry) KindNodeType(kind string) (string, bool) {
	v, ok := r.kinds[kind]
	return v, ok
}

func (r stubRegistry) HasRouteKeyField(code string) bool {
	_, ok := r.fields[code]
	return ok
}

func testRegistry() stubRegistry {
	return stubRegistry{
		nodeTypes: map[string]struct{}{"game": {}},
		kinds:     map[string]string{"player": "game"},
		fields:    map[string]struct{}{"player_id": {}},
	}
}

func TestDocumentRejectsMultipleKinds(t *testing.T) {
	doc := &source.Document{
		Version: 1,
		Scope:   "qa",
		Routes: []source.Route{{
			Kinds: source.Kinds{Kinds: []string{"player", "mail"}},
			Nodes: source.Nodes{Nodes: []source.Node{{
				NodeID: "n1",
				Type:   "game",
				RouteKeys: source.RouteKeys{
					Keys: source.Keys{Keys: []int32{1}},
				},
			}}},
		}},
	}
	if err := Document(doc, testRegistry()); err == nil || !strings.Contains(err.Error(), "exactly one kind") {
		t.Fatalf("expected single kind error, got %v", err)
	}
}

func TestDocumentRejectsKindNodeTypeMismatch(t *testing.T) {
	doc := &source.Document{
		Version: 1,
		Scope:   "qa",
		Routes: []source.Route{{
			Kinds: source.Kinds{Kinds: []string{"player"}},
			Nodes: source.Nodes{Nodes: []source.Node{{
				NodeID: "n1",
				Type:   "mail",
				RouteKeys: source.RouteKeys{
					Field: "player_id",
					Keys:  source.Keys{Keys: []int32{1}},
				},
			}}},
		}},
	}
	if err := Document(doc, testRegistry()); err == nil || !strings.Contains(err.Error(), "requires node_type") {
		t.Fatalf("expected kind/node_type mismatch error, got %v", err)
	}
}

func TestRouteKeyOverlapByNodeType(t *testing.T) {
	doc := &source.Document{
		Version: 1,
		Scope:   "qa",
		Routes: []source.Route{
			{
				Kinds: source.Kinds{Kinds: []string{"player"}},
				Nodes: source.Nodes{Nodes: []source.Node{{
					NodeID: "n1",
					Type:   "game",
					RouteKeys: source.RouteKeys{
						Keys: source.Keys{Keys: []int32{1001}},
					},
				}}},
			},
			{
				Kinds: source.Kinds{Kinds: []string{"player"}},
				Nodes: source.Nodes{Nodes: []source.Node{{
					NodeID: "n2",
					Type:   "game",
					RouteKeys: source.RouteKeys{
						Keys: source.Keys{Keys: []int32{1001}},
					},
				}}},
			},
		},
	}
	if err := Document(doc, testRegistry()); err == nil || !strings.Contains(err.Error(), "route_key_overlap") {
		t.Fatalf("expected overlap error, got %v", err)
	}
}

func TestRouteKeyOverlapRangeAndKey(t *testing.T) {
	doc := &source.Document{
		Version: 1,
		Scope:   "qa",
		Routes: []source.Route{
			{
				Kinds: source.Kinds{Kinds: []string{"player"}},
				Nodes: source.Nodes{Nodes: []source.Node{{
					NodeID: "n1",
					Type:   "game",
					RouteKeys: source.RouteKeys{
						Ranges: source.Ranges{Ranges: []source.Range{{Start: 2000, End: 2099}}},
					},
				}}},
			},
			{
				Kinds: source.Kinds{Kinds: []string{"player"}},
				Nodes: source.Nodes{Nodes: []source.Node{{
					NodeID: "n2",
					Type:   "game",
					RouteKeys: source.RouteKeys{
						Keys: source.Keys{Keys: []int32{2005}},
					},
				}}},
			},
		},
	}
	if err := RouteKeyOverlapByNodeType(doc); err == nil || !strings.Contains(err.Error(), "route_key_overlap") {
		t.Fatalf("expected range/key overlap error, got %v", err)
	}
}
