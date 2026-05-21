package staticrouter

import (
	"bytes"
	"context"
	"fmt"
	"sync/atomic"

	goredis "github.com/redis/go-redis/v9"
	"staticrouter/model"
	redisstore "staticrouter/store/redis"
)

type RouteContext = model.RouteContext
type RouteRecord = model.RouteRecord
type RouteSnapshot = model.RouteSnapshot
type RouteEvent = model.RouteEvent
type RouteEventType = model.RouteEventType

const (
	RouteEventType_ROUTE_EVENT_TYPE_UNSPECIFIED = model.RouteEventType_ROUTE_EVENT_TYPE_UNSPECIFIED
	RouteEventType_ROUTE_EVENT_TYPE_REPLACE_ALL = model.RouteEventType_ROUTE_EVENT_TYPE_REPLACE_ALL
)

type Config struct {
	Scope string
}

type RedisConfig struct {
	Host      string
	Password  string
	Index     int
	IsCluster bool
}

type SnapshotStore interface {
	GetSnapshot(ctx context.Context, scope string) (*model.RouteSnapshot, error)
	ReplaceSnapshot(ctx context.Context, snapshot *model.RouteSnapshot) error
	Watch(ctx context.Context, scope string) (<-chan *model.RouteSnapshot, error)
}

type Option func(*initOptions) error

type initOptions struct {
	ctx          context.Context
	scope        string
	redisConfig  *RedisConfig
	redisClient  goredis.UniversalClient
	store        SnapshotStore
	storeFactory func(RedisConfig) SnapshotStore
}

var defaultRouter atomic.Pointer[Router]
var defaultContext atomic.Pointer[context.Context]

func Init(opts ...Option) error {
	options := initOptions{
		ctx: context.Background(),
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(&options); err != nil {
			return err
		}
	}

	var store SnapshotStore
	switch {
	case options.store != nil:
		store = options.store
	case options.redisClient != nil:
		store = redisstore.NewWithUniversalClient(options.redisClient)
	case options.redisConfig != nil:
		if options.storeFactory != nil {
			store = options.storeFactory(*options.redisConfig)
		} else {
			store = redisstore.New(redisstore.Config{
				Host:      options.redisConfig.Host,
				Password:  options.redisConfig.Password,
				Index:     options.redisConfig.Index,
				IsCluster: options.redisConfig.IsCluster,
			})
		}
	}

	router := NewRouterWithConfig(Config{Scope: options.scope}, store)

	if store != nil {
		if err := router.Start(options.ctx); err != nil {
			return err
		}
	}

	defaultRouter.Store(router)
	defaultContext.Store(&options.ctx)
	return nil
}

func GetRoute(routeCtx *RouteContext) (*RouteRecord, bool) {
	router := defaultRouter.Load()
	if router == nil {
		return nil, false
	}
	return router.Get(routeCtx)
}

func UpdateConfig(mode ConfigMode, content []byte) error {
	router := defaultRouter.Load()
	if router == nil {
		return fmt.Errorf("staticrouter: not initialized")
	}
	snapshot, err := LoadRouteSnapshot(mode, bytes.NewReader(content))
	if err != nil {
		return err
	}
	scope := router.Config().Scope
	if scope != "" {
		if snapshot.GetScope() == "" {
			snapshot.Scope = scope
		} else if snapshot.GetScope() != scope {
			return fmt.Errorf("staticrouter: scope mismatch, option=%s snapshot=%s", scope, snapshot.GetScope())
		}
	}
	ctx := context.Background()
	if stored := defaultContext.Load(); stored != nil && *stored != nil {
		ctx = *stored
	}
	return router.ReplaceAll(ctx, snapshot)
}

func WithContext(ctx context.Context) Option {
	return func(o *initOptions) error {
		if ctx == nil {
			return fmt.Errorf("staticrouter: context is nil")
		}
		o.ctx = ctx
		return nil
	}
}

func WithScope(scope string) Option {
	return func(o *initOptions) error {
		o.scope = scope
		return nil
	}
}

func WithRedisConfig(cfg RedisConfig) Option {
	return func(o *initOptions) error {
		o.redisConfig = &cfg
		return nil
	}
}

func WithRedisInstance(client goredis.UniversalClient) Option {
	return func(o *initOptions) error {
		if client == nil {
			return fmt.Errorf("staticrouter: redis instance is nil")
		}
		o.redisClient = client
		return nil
	}
}

func WithStore(store SnapshotStore) Option {
	return func(o *initOptions) error {
		o.store = store
		return nil
	}
}

func WithStoreFactory(factory func(RedisConfig) SnapshotStore) Option {
	return func(o *initOptions) error {
		o.storeFactory = factory
		return nil
	}
}
