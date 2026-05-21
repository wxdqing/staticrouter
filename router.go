package staticrouter

import (
	"context"
	"errors"
	"sort"
	"sync/atomic"
)

var ErrRouterAlreadyStarted = errors.New("staticrouter: router already started")

type Router struct {
	config  Config
	store   SnapshotStore
	table   atomic.Pointer[runtimeTable]
	started atomic.Bool
}

func NewRouter(s SnapshotStore) *Router {
	return NewRouterWithConfig(Config{}, s)
}

func NewRouterWithConfig(cfg Config, s SnapshotStore) *Router {
	router := &Router{
		config: cfg,
		store:  s,
	}
	router.table.Store(&runtimeTable{
		exact:  make(map[routeMapKey]*RouteRecord),
		ranges: make(map[routeGroupKey][]*RouteRecord),
	})
	return router
}

func (r *Router) Get(routeCtx *RouteContext) (*RouteRecord, bool) {
	current := r.table.Load()
	if routeCtx == nil || current == nil {
		return nil, false
	}

	if route, ok := current.exact[newRouteMapKey(routeCtx.GetKind(), routeCtx.GetNodeType(), routeCtx.GetRouteKey())]; ok {
		return route, true
	}

	routes := current.ranges[newRouteGroupKey(routeCtx.GetKind(), routeCtx.GetNodeType())]
	if len(routes) == 0 {
		return nil, false
	}

	idx := sort.Search(len(routes), func(i int) bool {
		return routes[i].GetRouteKeyStart() > routeCtx.GetRouteKey()
	})
	if idx == 0 {
		return nil, false
	}
	candidate := routes[idx-1]
	if routeCtx.GetRouteKey() >= candidate.GetRouteKeyStart() &&
		routeCtx.GetRouteKey() <= candidate.GetRouteKeyEnd() {
		return candidate, true
	}
	return nil, false
}

func (r *Router) ReplaceAll(ctx context.Context, snapshot *RouteSnapshot) error {
	normalized, err := NormalizeSnapshot(snapshot)
	if err != nil {
		return err
	}
	compiled, err := compileSnapshot(normalized)
	if err != nil {
		return err
	}
	if r.store != nil {
		if err := r.store.ReplaceSnapshot(ctx, normalized); err != nil {
			return err
		}
	}
	r.table.Store(compiled)
	return nil
}

func (r *Router) ReplaceAllFromFile(ctx context.Context, path string) error {
	snapshot, err := LoadRouteSnapshotFromFile(path)
	if err != nil {
		return err
	}
	return r.ReplaceAll(ctx, snapshot)
}

func (r *Router) Start(ctx context.Context) error {
	if r.store == nil {
		return nil
	}
	if !r.started.CompareAndSwap(false, true) {
		return ErrRouterAlreadyStarted
	}
	snapshot, err := r.store.GetSnapshot(ctx, r.config.Scope)
	if err != nil {
		r.started.Store(false)
		return err
	}
	normalized, err := NormalizeSnapshot(snapshot)
	if err != nil {
		r.started.Store(false)
		return err
	}
	compiled, err := compileSnapshot(normalized)
	if err != nil {
		r.started.Store(false)
		return err
	}
	r.table.Store(compiled)

	go func() {
		for {
			ch, err := r.store.Watch(ctx, r.config.Scope)
			if err != nil {
				return
			}
			select {
			case <-ctx.Done():
				return
			default:
			}

			reload := false
			for !reload {
				select {
				case <-ctx.Done():
					return
				case snapshot, ok := <-ch:
					if !ok {
						reload = true
						break
					}
					normalized, err := NormalizeSnapshot(snapshot)
					if err != nil {
						continue
					}
					compiled, err := compileSnapshot(normalized)
					if err != nil {
						continue
					}
					current := r.table.Load()
					if current == nil || normalized.GetVersion() >= current.version {
						r.table.Store(compiled)
					}
				}
			}

			snapshot, err := r.store.GetSnapshot(ctx, r.config.Scope)
			if err != nil {
				continue
			}
			normalized, err := NormalizeSnapshot(snapshot)
			if err != nil {
				continue
			}
			compiled, err := compileSnapshot(normalized)
			if err != nil {
				continue
			}
			current := r.table.Load()
			if current == nil || normalized.GetVersion() >= current.version {
				r.table.Store(compiled)
			}
		}
	}()
	return nil
}

func (r *Router) Config() Config {
	return r.config
}
