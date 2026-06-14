// Package rediskeys defines canonical Redis key names for the staticrouter ecosystem.
//
// All consumers (runtime store, dashboard admin cache, ops tools) should import
// this package instead of hard-coding key strings.
package rediskeys

import "strings"

const (
	// Prefix is the root namespace for every staticrouter Redis key.
	Prefix = "staticrouter"

	// SnapshotPrefix stores compiled runtime RouteSnapshot protobuf (String).
	SnapshotPrefix = Prefix + ":snapshot"

	// EventsPrefix stores publish notifications (Stream, field "snapshot").
	EventsPrefix = Prefix + ":events"

	// AdminTablePrefix stores dashboard WebUI RouteTable cache protobuf (String).
	// Distinct from SnapshotPrefix: admin holds raw JSON content + metadata for editing.
	AdminTablePrefix = Prefix + ":admin"
)

// ScopeTag returns the Redis cluster hash-tag for a scope, e.g. "{dev}".
func ScopeTag(scope string) string {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return "{}"
	}
	return "{" + scope + "}"
}

// SnapshotKey is the runtime snapshot string key: staticrouter:snapshot:{scope}.
func SnapshotKey(scope string) string {
	return SnapshotPrefix + ":" + ScopeTag(scope)
}

// SnapshotMetaKey holds snapshot version metadata (Hash, field "version").
func SnapshotMetaKey(scope string) string {
	return SnapshotKey(scope) + ":meta"
}

// EventsKey is the runtime change stream: staticrouter:events:{scope}.
func EventsKey(scope string) string {
	return EventsPrefix + ":" + ScopeTag(scope)
}

// AdminTableKey is the dashboard management-plane cache: staticrouter:admin:{scope}.
func AdminTableKey(scope string) string {
	return AdminTablePrefix + ":" + ScopeTag(scope)
}

// RuntimeKeysForScope returns runtime publish/subscribe keys (excludes admin cache).
func RuntimeKeysForScope(scope string) []string {
	return []string{
		SnapshotKey(scope),
		SnapshotMetaKey(scope),
		EventsKey(scope),
	}
}

// KeysForScope returns all keys owned by a scope (useful for purge/diagnostics).
func KeysForScope(scope string) []string {
	keys := RuntimeKeysForScope(scope)
	keys = append(keys, AdminTableKey(scope))
	return keys
}
