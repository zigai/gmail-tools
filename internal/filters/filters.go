// Package filters turns Gmail filters into search queries and readable actions,
// and drafts gmailctl rules for senders that no filter handles.
package filters

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zigai/gmail-tools/internal/gmail"
	"github.com/zigai/gmail-tools/internal/senders"
)

const (
	labelInbox     = "INBOX"
	labelUnread    = "UNREAD"
	labelTrash     = "TRASH"
	labelSpam      = "SPAM"
	labelStarred   = "STARRED"
	labelImportant = "IMPORTANT"
)

var (
	addedLabelActions = map[string]string{
		labelTrash:     "delete",
		labelSpam:      "spam",
		labelStarred:   "star",
		labelImportant: "important",
	}
	removedLabelActions = map[string]string{
		labelInbox:     "archive",
		labelUnread:    "mark read",
		labelSpam:      "never spam",
		labelImportant: "not important",
	}
)

// Query returns the Gmail search query that matches the same messages as a
// filter's criteria.
func Query(c gmail.Criteria) string {
	var parts []string

	add := func(operator, value string) {
		if value != "" {
			parts = append(parts, operator+"("+value+")")
		}
	}

	add("from:", c.From)
	add("to:", c.To)
	add("subject:", c.Subject)
	add("", c.Query)
	add("-", c.NegatedQuery)

	if c.HasAttachment {
		parts = append(parts, "has:attachment")
	}

	switch c.SizeComparison {
	case "larger":
		parts = append(parts, "larger:"+strconv.FormatInt(c.Size, 10))
	case "smaller":
		parts = append(parts, "smaller:"+strconv.FormatInt(c.Size, 10))
	}

	return strings.Join(parts, " ")
}

// Describe lists a filter's actions in words, using labelNames to name user labels.
func Describe(a gmail.Action, labelNames map[string]string) string {
	var parts []string

	for _, id := range a.AddLabelIDs {
		action, ok := addedLabelActions[id]
		if !ok {
			action = "label " + labelName(id, labelNames)
		}

		parts = append(parts, action)
	}

	for _, id := range a.RemoveLabelIDs {
		action, ok := removedLabelActions[id]
		if !ok {
			action = "remove " + labelName(id, labelNames)
		}

		parts = append(parts, action)
	}

	if a.Forward != "" {
		parts = append(parts, "forward to "+a.Forward)
	}

	return strings.Join(parts, ", ")
}

// Suggest drafts one gmailctl rule per sender that archives and labels its mail.
// Senders on a mailing list are matched by list, the rest by address.
func Suggest(counts []senders.Sender) string {
	var b strings.Builder

	for _, sender := range counts {
		criterion, label := "from: "+jsonnetString(sender.Address), senderLabel(sender.Address)
		if sender.ListID != "" {
			id := listID(sender.ListID)
			name, _, _ := strings.Cut(id, ".")
			criterion, label = "list: "+jsonnetString(id), "Lists/"+name
		}

		fmt.Fprintf(&b, "// %d messages, %d unread\n", sender.Messages, sender.Unread)
		fmt.Fprintf(&b, "{\n  filter: { %s },\n", criterion)
		fmt.Fprintf(&b, "  actions: { archive: true, labels: [%s] },\n},\n", jsonnetString(label))
	}

	return b.String()
}

func jsonnetString(value string) string {
	return "'" + strings.NewReplacer(`\`, `\\`, "'", `\'`, "\n", " ").Replace(value) + "'"
}

func labelName(id string, labelNames map[string]string) string {
	if name, ok := labelNames[id]; ok {
		return name
	}

	return id
}

func listID(header string) string {
	if start := strings.LastIndex(header, "<"); start >= 0 {
		if end := strings.Index(header[start:], ">"); end > 0 {
			return header[start+1 : start+end]
		}
	}

	return strings.TrimSpace(header)
}

func senderLabel(address string) string {
	_, domain, ok := strings.Cut(address, "@")
	if !ok {
		return "Senders/" + address
	}

	if dot := strings.LastIndex(domain, "."); dot > 0 {
		domain = domain[:dot]
	}

	return "Senders/" + domain[strings.LastIndex(domain, ".")+1:]
}
