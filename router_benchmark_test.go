package staticrouter

import (
	"context"
	"fmt"
	"testing"
)

func BenchmarkRouterGetComplexTable(b *testing.B) {
	router := NewRouter(nil)
	snapshot := buildBenchmarkSnapshot()
	if err := router.ReplaceAll(context.Background(), snapshot); err != nil {
		b.Fatalf("replace all returned error: %v", err)
	}

	benchmarks := []struct {
		name    string
		queries []*RouteContext
	}{
		{
			name: "exact",
			queries: []*RouteContext{
				{Kind: "kind-00", NodeType: "node-type-00", RouteKey: 1},
				{Kind: "kind-09", NodeType: "node-type-04", RouteKey: 900000 + 40000 + 777},
				{Kind: "kind-17", NodeType: "node-type-02", RouteKey: 1700000 + 20000 + 1599},
			},
		},
		{
			name: "range",
			queries: []*RouteContext{
				{Kind: "kind-00", NodeType: "node-type-00", RouteKey: 2000010},
				{Kind: "kind-09", NodeType: "node-type-04", RouteKey: 2000000 + 900000 + 40000 + 1205},
				{Kind: "kind-17", NodeType: "node-type-02", RouteKey: 2000000 + 1700000 + 20000 + 7999},
			},
		},
		{
			name: "miss",
			queries: []*RouteContext{
				{Kind: "kind-00", NodeType: "node-type-00", RouteKey: 1999999},
				{Kind: "kind-09", NodeType: "node-type-04", RouteKey: 3999999},
				{Kind: "unknown", NodeType: "node-type-00", RouteKey: 1},
			},
		},
		{
			name: "mixed",
			queries: []*RouteContext{
				{Kind: "kind-00", NodeType: "node-type-00", RouteKey: 1},
				{Kind: "kind-09", NodeType: "node-type-04", RouteKey: 2000000 + 900000 + 40000 + 1205},
				{Kind: "kind-17", NodeType: "node-type-02", RouteKey: 3999999},
				{Kind: "kind-17", NodeType: "node-type-02", RouteKey: 1700000 + 20000 + 1599},
			},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if route, _ := router.Get(bm.queries[i%len(bm.queries)]); route != nil {
					benchmarkRouteSink = route
				}
			}
		})
	}
}

var benchmarkRouteSink *RouteRecord

func buildBenchmarkSnapshot() *RouteSnapshot {
	const (
		kindCount     = 18
		nodeTypeCount = 5
		exactPerGroup = 1600
		rangePerGroup = 800
	)

	routes := make([]*RouteRecord, 0, kindCount*nodeTypeCount*(exactPerGroup+rangePerGroup))
	for kindIndex := 0; kindIndex < kindCount; kindIndex++ {
		for nodeTypeIndex := 0; nodeTypeIndex < nodeTypeCount; nodeTypeIndex++ {
			kind := fmt.Sprintf("kind-%02d", kindIndex)
			nodeType := fmt.Sprintf("node-type-%02d", nodeTypeIndex)
			groupBase := int32(kindIndex*100000 + nodeTypeIndex*10000)

			for exactIndex := 0; exactIndex < exactPerGroup; exactIndex++ {
				routeKey := groupBase + int32(exactIndex+1)
				routes = append(routes, &RouteRecord{
					Kind:      kind,
					NodeType:  nodeType,
					RouteKeys: []int32{routeKey},
					NodeId:    fmt.Sprintf("%s-%s-exact-%04d", kind, nodeType, exactIndex),
					Metadata: map[string]string{
						"kind":      kind,
						"node_type": nodeType,
					},
				})
			}

			rangeBase := int32(2000000) + groupBase
			for rangeIndex := 0; rangeIndex < rangePerGroup; rangeIndex++ {
				start := rangeBase + int32(rangeIndex*10)
				routes = append(routes, &RouteRecord{
					Kind:          kind,
					NodeType:      nodeType,
					RouteKeyStart: start,
					RouteKeyEnd:   start + 9,
					NodeId:        fmt.Sprintf("%s-%s-range-%04d", kind, nodeType, rangeIndex),
					Metadata: map[string]string{
						"kind":      kind,
						"node_type": nodeType,
					},
				})
			}
		}
	}

	return &RouteSnapshot{
		Version: 1,
		Scope:   "bench",
		Routes:  routes,
	}
}
