package rediskeys

import "testing"

func TestScopeTag(t *testing.T) {
	if got := ScopeTag("dev"); got != "{dev}" {
		t.Fatalf("ScopeTag(dev) = %q, want {dev}", got)
	}
	if got := ScopeTag("  qa  "); got != "{qa}" {
		t.Fatalf("ScopeTag trimmed = %q, want {qa}", got)
	}
}

func TestRuntimeKeys(t *testing.T) {
	scope := "qa"
	if got, want := SnapshotKey(scope), "staticrouter:snapshot:{qa}"; got != want {
		t.Fatalf("SnapshotKey = %q, want %q", got, want)
	}
	if got, want := SnapshotMetaKey(scope), "staticrouter:snapshot:{qa}:meta"; got != want {
		t.Fatalf("SnapshotMetaKey = %q, want %q", got, want)
	}
	if got, want := EventsKey(scope), "staticrouter:events:{qa}"; got != want {
		t.Fatalf("EventsKey = %q, want %q", got, want)
	}
}

func TestAdminTableKey(t *testing.T) {
	if got, want := AdminTableKey("dev"), "staticrouter:admin:{dev}"; got != want {
		t.Fatalf("AdminTableKey = %q, want %q", got, want)
	}
}

func TestRuntimeKeysForScope(t *testing.T) {
	keys := RuntimeKeysForScope("prod")
	if len(keys) != 3 {
		t.Fatalf("expected 3 runtime keys, got %d", len(keys))
	}
	if keys[0] != "staticrouter:snapshot:{prod}" {
		t.Fatalf("unexpected snapshot key %q", keys[0])
	}
}

func TestKeysForScopeIncludesAdmin(t *testing.T) {
	if len(KeysForScope("dev")) != 4 {
		t.Fatalf("expected 4 keys including admin")
	}
}
