//go:build integration

package httpapi

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Library and search agree on "unknown": a clip without any identified voice, including
// a clip with no Local speaker at all (its search segments are unknown), is unknown; a
// clip whose voices are all identified is not.
func testLibraryUnknownFilterMatchesSearch(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 3)
	identified, mixed, undiarized := h.clipIDs[0], h.clipIDs[1], h.clipIDs[2]
	h.validate(h.fixtureSpeaker(identified, "ONLY", usableAt(0)), "Alice")
	h.validate(h.fixtureSpeaker(mixed, "KNOWN", usableAt(90)), "Bob")
	h.fixtureSpeaker(mixed, "STRANGER", usableAt(180))
	seen := h.visibleClips(h.adminUser(), "person_unknown=true")
	if seen[identified] || !seen[mixed] || !seen[undiarized] {
		t.Fatalf("unknown filter: identified=%v mixed=%v undiarized=%v", seen[identified], seen[mixed], seen[undiarized])
	}
}
