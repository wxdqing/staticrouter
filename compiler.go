package staticrouter

import (
	"errors"
	"sort"
)

var ErrRouteConflict = errors.New("staticrouter: route conflict")

type runtimeTable struct {
	version int64
	exact   map[routeMapKey]*RouteRecord
	ranges  map[routeGroupKey][]*RouteRecord
}

type routeMapKey struct {
	kind     string
	nodeType string
	routeKey int32
}

type routeGroupKey struct {
	kind     string
	nodeType string
}

func compileSnapshot(snapshot *RouteSnapshot) (*runtimeTable, error) {
	if snapshot == nil {
		return &runtimeTable{
			exact:  make(map[routeMapKey]*RouteRecord),
			ranges: make(map[routeGroupKey][]*RouteRecord),
		}, nil
	}

	compiled := &runtimeTable{
		version: snapshot.GetVersion(),
		exact:   make(map[routeMapKey]*RouteRecord),
		ranges:  make(map[routeGroupKey][]*RouteRecord),
	}

	for _, route := range snapshot.GetRoutes() {
		if route == nil {
			continue
		}
		if err := validateRouteRecord(route); err != nil {
			return nil, err
		}
		for _, routeKey := range route.GetRouteKeys() {
			key := newRouteMapKey(route.GetKind(), route.GetNodeType(), routeKey)
			if _, exists := compiled.exact[key]; exists {
				return nil, ErrRouteConflict
			}
			for _, rangeRoute := range compiled.ranges[newRouteGroupKey(route.GetKind(), route.GetNodeType())] {
				if sameKindAndType(rangeRoute, route) &&
					routeKey >= rangeRoute.GetRouteKeyStart() &&
					routeKey <= rangeRoute.GetRouteKeyEnd() {
					return nil, ErrRouteConflict
				}
			}
			compiled.exact[key] = route
		}

		if route.GetRouteKeyStart() != 0 || route.GetRouteKeyEnd() != 0 {
			for _, existing := range compiled.exact {
				if !sameKindAndType(existing, route) {
					continue
				}
				for _, routeKey := range existing.GetRouteKeys() {
					if routeKey >= route.GetRouteKeyStart() && routeKey <= route.GetRouteKeyEnd() {
						return nil, ErrRouteConflict
					}
				}
			}
			for _, existing := range compiled.ranges[newRouteGroupKey(route.GetKind(), route.GetNodeType())] {
				if route.GetRouteKeyStart() <= existing.GetRouteKeyEnd() &&
					existing.GetRouteKeyStart() <= route.GetRouteKeyEnd() {
					return nil, ErrRouteConflict
				}
			}
			groupKey := newRouteGroupKey(route.GetKind(), route.GetNodeType())
			compiled.ranges[groupKey] = append(compiled.ranges[groupKey], route)
		}
	}

	for groupKey := range compiled.ranges {
		sort.Slice(compiled.ranges[groupKey], func(i, j int) bool {
			return compiled.ranges[groupKey][i].GetRouteKeyStart() < compiled.ranges[groupKey][j].GetRouteKeyStart()
		})
	}

	return compiled, nil
}

func validateRouteRecord(route *RouteRecord) error {
	if route == nil {
		return nil
	}
	hasKeys := len(route.GetRouteKeys()) > 0
	hasRange := route.GetRouteKeyStart() != 0 || route.GetRouteKeyEnd() != 0
	if !hasKeys && !hasRange {
		return errors.New("staticrouter: route record must contain route keys or range")
	}
	if hasRange && route.GetRouteKeyStart() > route.GetRouteKeyEnd() {
		return errors.New("staticrouter: invalid route range")
	}
	return nil
}

func sameKindAndType(a *RouteRecord, b *RouteRecord) bool {
	return a.GetKind() == b.GetKind() && a.GetNodeType() == b.GetNodeType()
}

func newRouteMapKey(kind string, nodeType string, routeKey int32) routeMapKey {
	return routeMapKey{kind: kind, nodeType: nodeType, routeKey: routeKey}
}

func newRouteGroupKey(kind string, nodeType string) routeGroupKey {
	return routeGroupKey{kind: kind, nodeType: nodeType}
}
