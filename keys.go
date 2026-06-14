package staticrouter

import "github.com/wxdqing/staticrouter/rediskeys"

// Redis key helpers re-exported from rediskeys for convenience.
// Prefer importing this package when already depending on staticrouter.

const (
	RedisPrefix            = rediskeys.Prefix
	RedisSnapshotPrefix    = rediskeys.SnapshotPrefix
	RedisEventsPrefix      = rediskeys.EventsPrefix
	RedisAdminTablePrefix  = rediskeys.AdminTablePrefix
)

func RedisScopeTag(scope string) string { return rediskeys.ScopeTag(scope) }

func SnapshotKey(scope string) string { return rediskeys.SnapshotKey(scope) }

func SnapshotMetaKey(scope string) string { return rediskeys.SnapshotMetaKey(scope) }

func EventsKey(scope string) string { return rediskeys.EventsKey(scope) }

func AdminTableKey(scope string) string { return rediskeys.AdminTableKey(scope) }

func RuntimeRedisKeysForScope(scope string) []string { return rediskeys.RuntimeKeysForScope(scope) }

func RedisKeysForScope(scope string) []string { return rediskeys.KeysForScope(scope) }
