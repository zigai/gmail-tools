package gmail_test

import (
	"bytes"
	"slices"
	"testing"

	"github.com/zigai/gmail-tools/internal/gmail"
	"github.com/zigai/gmail-tools/internal/gog"
	"github.com/zigai/gmail-tools/internal/process"
	"github.com/zigai/gmail-tools/internal/testutil"
)

func newClient(fake testutil.FakeBinary) gmail.Client {
	return gmail.Client{
		Runner: gog.Runner{Binary: fake.Path, Streams: process.Streams{Stderr: &bytes.Buffer{}}},
		Email:  "user@example.com",
	}
}

func pagedGog(t *testing.T) testutil.FakeBinary {
	t.Helper()

	return testutil.NewFakeBinary(
		t,
		testutil.Response{Pattern: "--page=second", Stdout: `{"messages": [{"id": "m3"}]}`},
		testutil.Response{
			Stdout: `{"messages": [{"id": "m1"}, {"id": "m2"}], "nextPageToken": "second"}`,
		},
	)
}

func ids(messages []gmail.Message) []string {
	out := make([]string, len(messages))
	for i, m := range messages {
		out[i] = m.ID
	}

	return out
}

func TestSearchFollowsPagesUntilTheEnd(t *testing.T) {
	fake := pagedGog(t)

	messages, err := newClient(fake).Search(t.Context(), "in:inbox", 0)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if got := ids(messages); !slices.Equal(got, []string{"m1", "m2", "m3"}) {
		t.Fatalf("Search() ids = %q, want all three pages' messages", got)
	}
}

func TestSearchStopsAtLimit(t *testing.T) {
	fake := pagedGog(t)

	messages, err := newClient(fake).Search(t.Context(), "in:inbox", 2)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if got := ids(messages); !slices.Equal(got, []string{"m1", "m2"}) {
		t.Fatalf("Search() ids = %q, want the first two", got)
	}

	if calls := fake.Calls(t); len(calls) != 1 || !slices.Contains(calls[0], "--max=2") {
		t.Fatalf("gog calls = %q, want one call asking for 2", calls)
	}
}

func TestCountReportsLowerBounds(t *testing.T) {
	fake := testutil.NewFakeBinary(t, testutil.Response{Stdout: `{"totalMatchesAtLeast": 500}`})

	count, err := newClient(fake).Count(t.Context(), "from:a@example.com")
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}

	if count != (gmail.Count{Total: 500, Exact: false}) {
		t.Fatalf("Count() = %+v, want a lower bound of 500", count)
	}
}
