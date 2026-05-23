package redis

import (
	"context"
	"errors"
	"fmt"

	goredis "github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"

	"staticrouter/model"
)

const (
	snapshotKeyPrefix = "staticrouter:snapshot"
	streamKeyPrefix   = "staticrouter:events"
	consumerBlock     = 1000
	initialStreamID   = "0-0"
)

var replaceSnapshotScript = goredis.NewScript(`
local snapshot_key = KEYS[1]
local stream_key = KEYS[2]
local meta_key = KEYS[3]
local new_version = tonumber(ARGV[1])
local new_data = ARGV[2]

local current = redis.call("GET", snapshot_key)
if current then
  local current_version = tonumber(redis.call("HGET", meta_key, "version"))
  if current_version and new_version < current_version then
    return 0
  end
end

redis.call("SET", snapshot_key, new_data)
redis.call("HSET", meta_key, "version", tostring(new_version))
redis.call("XADD", stream_key, "*", "snapshot", new_data)
return 1
`)

type Store struct {
	client goredis.UniversalClient
}

func New(cfg Config) *Store {
	return NewWithUniversalClient(NewClient(cfg))
}

func NewWithUniversalClient(client goredis.UniversalClient) *Store {
	return &Store{
		client: client,
	}
}

func (s *Store) GetSnapshot(ctx context.Context, scope string) (*model.RouteSnapshot, error) {
	data, err := s.client.Get(ctx, snapshotKey(scope)).Bytes()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	snapshot := &model.RouteSnapshot{}
	if err := proto.Unmarshal(data, snapshot); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (s *Store) ReplaceSnapshot(ctx context.Context, snapshot *model.RouteSnapshot) error {
	if snapshot == nil {
		return nil
	}

	data, err := proto.Marshal(snapshot)
	if err != nil {
		return err
	}

	result, err := replaceSnapshotScript.Run(ctx, s.client,
		[]string{snapshotKey(snapshot.GetScope()), streamKey(snapshot.GetScope()), snapshotMetaKey(snapshot.GetScope())},
		snapshot.GetVersion(),
		data,
	).Int()
	if err != nil {
		return err
	}
	if result == 0 {
		return fmt.Errorf("staticrouter: snapshot version rollback")
	}
	return nil
}

func (s *Store) Watch(ctx context.Context, scope string) (<-chan *model.RouteSnapshot, error) {
	out := make(chan *model.RouteSnapshot)
	go func() {
		defer close(out)
		streamID := initialStreamID
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			streams, err := s.client.XRead(ctx, &goredis.XReadArgs{
				Streams: []string{streamKey(scope), streamID},
				Block:   consumerBlock,
				Count:   10,
			}).Result()
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return
				}
				if err == goredis.Nil {
					continue
				}
				return
			}

			for _, stream := range streams {
				for _, message := range stream.Messages {
					raw, ok := message.Values["snapshot"]
					if !ok {
						continue
					}
					data, ok := raw.(string)
					if !ok {
						if bytes, byteOK := raw.([]byte); byteOK {
							data = string(bytes)
						} else {
							continue
						}
					}
					snapshot := &model.RouteSnapshot{}
					if err := proto.Unmarshal([]byte(data), snapshot); err != nil {
						continue
					}
					streamID = message.ID
					select {
					case out <- snapshot:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()
	return out, nil
}

func snapshotKey(scope string) string {
	return snapshotKeyPrefix + ":{" + scope + "}"
}

func snapshotMetaKey(scope string) string {
	return snapshotKey(scope) + ":meta"
}

func streamKey(scope string) string {
	return streamKeyPrefix + ":{" + scope + "}"
}
