// Package senders counts messages per sender and mailing list.
package senders

import (
	"cmp"
	"net/mail"
	"slices"
	"strings"

	"github.com/zigai/gmail-tools/internal/gmail"
)

const (
	// ListIDHeader is the header to request from a search so Count can group
	// mailing lists.
	ListIDHeader = "List-Id"
	unreadLabel  = "UNREAD"
)

type Sender struct {
	Address  string `json:"address"`
	Name     string `json:"name,omitempty"`
	ListID   string `json:"list_id,omitempty"`
	Messages int    `json:"messages"`
	Unread   int    `json:"unread"`
}

// Count groups messages by sender address and mailing list, ordered by message
// count, then unread count, then address.
func Count(messages []gmail.Message) []Sender {
	type key struct{ address, listID string }

	byKey := map[key]*Sender{}

	for _, msg := range messages {
		address, name := parseFrom(msg.From)
		listID := strings.TrimSpace(msg.Headers[ListIDHeader])
		k := key{address: address, listID: listID}

		sender, ok := byKey[k]
		if !ok {
			sender = &Sender{Address: address, Name: name, ListID: listID, Messages: 0, Unread: 0}
			byKey[k] = sender
		}

		sender.Messages++
		if slices.Contains(msg.Labels, unreadLabel) {
			sender.Unread++
		}
	}

	out := make([]Sender, 0, len(byKey))
	for _, sender := range byKey {
		out = append(out, *sender)
	}

	slices.SortFunc(out, func(a, b Sender) int {
		return cmp.Or(
			cmp.Compare(b.Messages, a.Messages),
			cmp.Compare(b.Unread, a.Unread),
			strings.Compare(a.Address, b.Address),
			strings.Compare(a.ListID, b.ListID),
		)
	})

	return out
}

func parseFrom(from string) (string, string) {
	addr, err := mail.ParseAddress(from)
	if err != nil {
		return strings.ToLower(strings.TrimSpace(from)), ""
	}

	return strings.ToLower(addr.Address), addr.Name
}
