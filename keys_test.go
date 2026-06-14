package staticrouter

import (
	"testing"

	"github.com/wxdqing/staticrouter/rediskeys"
)

func TestReexportedRedisKeysMatchRediskeysPackage(t *testing.T) {
	scope := "dev"
	if got, want := SnapshotKey(scope), rediskeys.SnapshotKey(scope); got != want {
		t.Fatalf("SnapshotKey = %q, want %q", got, want)
	}
	if got, want := SnapshotMetaKey(scope), rediskeys.SnapshotMetaKey(scope); got != want {
		t.Fatalf("SnapshotMetaKey = %q, want %q", got, want)
	}
	if got, want := EventsKey(scope), rediskeys.EventsKey(scope); got != want {
		t.Fatalf("EventsKey = %q, want %q", got, want)
	}
	if got, want := AdminTableKey(scope), rediskeys.AdminTableKey(scope); got != want {
		t.Fatalf("AdminTableKey = %q, want %q", got, want)
	}
	if len(RuntimeRedisKeysForScope(scope)) != len(rediskeys.RuntimeKeysForScope(scope)) {
		t.Fatalf("RuntimeRedisKeysForScope length mismatch")
	}
}
