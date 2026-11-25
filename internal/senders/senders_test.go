package senders_test

import (
	"slices"
	"testing"

	"github.com/zigai/gmail-tools/internal/gmail"
	"github.com/zigai/gmail-tools/internal/senders"
)

func TestCountGroupsAndOrdersSenders(t *testing.T) {
	daily := map[string]string{senders.ListIDHeader: "<daily.example.org>"}
	weekly := map[string]string{senders.ListIDHeader: "<weekly.example.org>"}
	messages := []gmail.Message{
		{From: "News Desk <News@Example.org>", Labels: []string{"INBOX", "UNREAD"}, Headers: daily},
		{From: "news@example.org", Labels: []string{"INBOX"}, Headers: daily},
		{
			From:    "News Desk <news@example.org>",
			Labels:  []string{"INBOX", "UNREAD"},
			Headers: weekly,
		},
		{From: "Ann <ann@example.com>", Labels: []string{"INBOX"}},
		{From: "not an address", Labels: []string{"INBOX", "UNREAD"}},
	}

	got := senders.Count(messages)

	want := []senders.Sender{
		{
			Address:  "news@example.org",
			Name:     "News Desk",
			ListID:   "<daily.example.org>",
			Messages: 2,
			Unread:   1,
		},
		{
			Address:  "news@example.org",
			Name:     "News Desk",
			ListID:   "<weekly.example.org>",
			Messages: 1,
			Unread:   1,
		},
		{Address: "not an address", Messages: 1, Unread: 1},
		{Address: "ann@example.com", Name: "Ann", Messages: 1, Unread: 0},
	}

	if !slices.Equal(got, want) {
		t.Fatalf("Count() =\n%+v\nwant\n%+v", got, want)
	}
}
