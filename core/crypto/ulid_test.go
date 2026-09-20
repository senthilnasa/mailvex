package crypto

import (
	"strings"
	"testing"
)

func TestPrefixedULIDs(t *testing.T) {
	prefixes := []string{
		PrefixOrg, PrefixTenant, PrefixUser, PrefixDomain,
		PrefixMessage, PrefixRecipient, PrefixAPIKey, PrefixWebhook,
	}

	for _, prefix := range prefixes {
		id := NewULID(prefix)
		if !strings.HasPrefix(id, prefix) {
			t.Fatalf("expected ID %s to start with prefix %s", id, prefix)
		}

		parsed, err := ParseULID(id, prefix)
		if err != nil {
			t.Fatalf("failed to parse ULID from %s: %v", id, err)
		}

		if parsed.Time() == 0 {
			t.Fatalf("parsed ULID has zero timestamp")
		}
	}
}
