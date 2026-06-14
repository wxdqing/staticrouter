package validate

import (
	"fmt"
	"strings"

	"github.com/wxdqing/staticrouter/source"
)

// Registry supplies registered codes for document validation.
type Registry interface {
	HasNodeType(code string) bool
	KindNodeType(kind string) (nodeType string, ok bool)
	HasRouteKeyField(code string) bool
}

// Document validates structure, registry references, and route-key overlap.
func Document(doc *source.Document, reg Registry) error {
	if doc == nil {
		return fmt.Errorf("staticrouter: document is nil")
	}
	if strings.TrimSpace(doc.Scope) == "" {
		return fmt.Errorf("staticrouter: scope is required")
	}
	if doc.Version < 1 {
		return fmt.Errorf("staticrouter: version must be >= 1")
	}
	for i := range doc.Routes {
		if err := route(&doc.Routes[i], reg); err != nil {
			return fmt.Errorf("staticrouter: route[%d]: %w", i, err)
		}
	}
	return RouteKeyOverlapByNodeType(doc)
}

func route(r *source.Route, reg Registry) error {
	if r == nil {
		return fmt.Errorf("route is nil")
	}
	if len(r.Kinds.Kinds) == 0 {
		return fmt.Errorf("kinds is empty")
	}
	if len(r.Kinds.Kinds) != 1 {
		return fmt.Errorf("exactly one kind per route is required")
	}
	kind := strings.TrimSpace(r.Kinds.Kinds[0])
	if kind == "" {
		return fmt.Errorf("kind is empty")
	}
	if reg != nil {
		if bound, ok := reg.KindNodeType(kind); ok {
			if bound == "" {
				return fmt.Errorf("kind %q has no bound node_type", kind)
			}
		}
	}
	if len(r.Nodes.Nodes) == 0 {
		return fmt.Errorf("nodes is empty")
	}
	seenNodeID := make(map[string]struct{}, len(r.Nodes.Nodes))
	for j := range r.Nodes.Nodes {
		if err := node(kind, &r.Nodes.Nodes[j], reg, seenNodeID); err != nil {
			return fmt.Errorf("node[%d]: %w", j, err)
		}
	}
	return nil
}

func node(kind string, n *source.Node, reg Registry, seenNodeID map[string]struct{}) error {
	if n == nil {
		return fmt.Errorf("node is nil")
	}
	nodeID := strings.TrimSpace(n.NodeID)
	if nodeID == "" {
		return fmt.Errorf("node_id is required")
	}
	if _, exists := seenNodeID[nodeID]; exists {
		return fmt.Errorf("duplicate node_id %q in route", nodeID)
	}
	seenNodeID[nodeID] = struct{}{}

	nodeType := strings.TrimSpace(n.Type)
	if nodeType == "" {
		nodeType = "game"
	}
	if reg != nil {
		if bound, ok := reg.KindNodeType(kind); ok && bound != nodeType {
			return fmt.Errorf("kind %q requires node_type %q, got %q", kind, bound, nodeType)
		}
		if !reg.HasNodeType(nodeType) {
			return fmt.Errorf("node_type %q is not registered", nodeType)
		}
		field := strings.TrimSpace(n.RouteKeys.Field)
		if field != "" && !reg.HasRouteKeyField(field) {
			return fmt.Errorf("route_key_field %q is not registered", field)
		}
	}

	hasKeys := len(n.RouteKeys.Keys.Keys) > 0
	hasRanges := len(n.RouteKeys.Ranges.Ranges) > 0
	if !hasKeys && !hasRanges {
		return fmt.Errorf("route_keys must contain keys or ranges")
	}
	if len(n.RouteKeys.Ranges.Ranges) > 1 {
		return fmt.Errorf("at most one range is allowed")
	}
	for _, rng := range n.RouteKeys.Ranges.Ranges {
		if rng.Start > rng.End {
			return fmt.Errorf("invalid range %d-%d", rng.Start, rng.End)
		}
	}
	return nil
}

// RouteKeyOverlapByNodeType rejects overlapping keys/ranges within the same node_type.
func RouteKeyOverlapByNodeType(doc *source.Document) error {
	if doc == nil {
		return nil
	}

	type interval struct {
		start  int32
		end    int32
		nodeID string
	}

	exact := make(map[string]map[int32]string) // nodeType -> key -> nodeID
	ranges := make(map[string][]interval)      // nodeType -> intervals

	for _, route := range doc.Routes {
		if len(route.Kinds.Kinds) == 0 {
			continue
		}
		for _, node := range route.Nodes.Nodes {
			nodeType := strings.TrimSpace(node.Type)
			if nodeType == "" {
				nodeType = "game"
			}
			nodeID := strings.TrimSpace(node.NodeID)

			if exact[nodeType] == nil {
				exact[nodeType] = make(map[int32]string)
			}
			for _, key := range node.RouteKeys.Keys.Keys {
				if owner, ok := exact[nodeType][key]; ok {
					return fmt.Errorf("staticrouter: route_key_overlap: node_type=%s key=%d nodes=%s,%s",
						nodeType, key, owner, nodeID)
				}
				exact[nodeType][key] = nodeID
				for _, iv := range ranges[nodeType] {
					if key >= iv.start && key <= iv.end {
						return fmt.Errorf("staticrouter: route_key_overlap: node_type=%s key=%d nodes=%s,%s",
							nodeType, key, nodeID, iv.nodeID)
					}
				}
			}

			if len(node.RouteKeys.Ranges.Ranges) == 1 {
				rng := node.RouteKeys.Ranges.Ranges[0]
				for key, owner := range exact[nodeType] {
					if key >= rng.Start && key <= rng.End {
						return fmt.Errorf("staticrouter: route_key_overlap: node_type=%s key=%d nodes=%s,%s",
							nodeType, key, owner, nodeID)
					}
				}
				for _, iv := range ranges[nodeType] {
					if rng.Start <= iv.end && iv.start <= rng.End {
						return fmt.Errorf("staticrouter: route_key_overlap: node_type=%s range=%d-%d nodes=%s,%s",
							nodeType, rng.Start, rng.End, nodeID, iv.nodeID)
					}
				}
				ranges[nodeType] = append(ranges[nodeType], interval{
					start:  rng.Start,
					end:    rng.End,
					nodeID: nodeID,
				})
			}
		}
	}

	return nil
}
