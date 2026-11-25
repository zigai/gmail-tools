package filters_test

import (
	"strings"
	"testing"

	"github.com/zigai/gmail-tools/internal/filters"
	"github.com/zigai/gmail-tools/internal/gmail"
	"github.com/zigai/gmail-tools/internal/senders"
)

func TestQueryCombinesEveryCriterion(t *testing.T) {
	got := filters.Query(gmail.Criteria{
		From:           "{a@example.com b@example.com}",
		To:             "me@example.com",
		Subject:        "\"weekly report\"",
		Query:          "has:userlabels",
		NegatedQuery:   "urgent",
		HasAttachment:  true,
		Size:           1000,
		SizeComparison: "larger",
	})

	want := "from:({a@example.com b@example.com}) to:(me@example.com) subject:(\"weekly report\") " +
		"(has:userlabels) -(urgent) has:attachment larger:1000"
	if got != want {
		t.Fatalf("Query() = %q, want %q", got, want)
	}
}

func TestQuerySkipsEmptyCriteria(t *testing.T) {
	if got := filters.Query(gmail.Criteria{From: "a@example.com"}); got != "from:(a@example.com)" {
		t.Fatalf("Query() = %q", got)
	}
}

func TestDescribeNamesActions(t *testing.T) {
	got := filters.Describe(gmail.Action{
		AddLabelIDs:    []string{"Label_1", "Label_404", "STARRED"},
		RemoveLabelIDs: []string{"INBOX", "UNREAD"},
		Forward:        "other@example.com",
	}, map[string]string{"Label_1": "Receipts"})

	want := "label Receipts, label Label_404, star, archive, mark read, forward to other@example.com"
	if got != want {
		t.Fatalf("Describe() = %q, want %q", got, want)
	}
}

func TestSuggestMatchesListsByListAndOthersBySender(t *testing.T) {
	got := filters.Suggest([]senders.Sender{
		{
			Address:  "bot@lists.example.org",
			ListID:   "Project news <news.lists.example.org>",
			Messages: 9,
			Unread:   4,
		},
		{Address: "o'brien@shop.example.co", Messages: 6, Unread: 6},
	})

	for _, want := range []string{
		"// 9 messages, 4 unread",
		"filter: { list: 'news.lists.example.org' }",
		"labels: ['Lists/news']",
		`filter: { from: 'o\'brien@shop.example.co' }`,
		"labels: ['Senders/example']",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("Suggest() =\n%s\nmissing %q", got, want)
		}
	}
}
