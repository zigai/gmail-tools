// Package gmail reads Gmail data for one account through gog's JSON output.
package gmail

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/zigai/gmail-tools/internal/gog"
)

const maxPageSize = 500

type Message struct {
	ID       string            `json:"id"`
	ThreadID string            `json:"thread_id"`
	Date     string            `json:"date"`
	From     string            `json:"from"`
	Subject  string            `json:"subject"`
	Labels   []string          `json:"labels"`
	Headers  map[string]string `json:"headers,omitempty"`
}

type gogMessage struct {
	ID       string            `json:"id"`
	ThreadID string            `json:"threadId"`        //nolint:tagliatelle // gog's output field
	Date     string            `json:"internalDateIso"` //nolint:tagliatelle // gog's output field
	From     string            `json:"from"`
	Subject  string            `json:"subject"`
	Labels   []string          `json:"labels"`
	Headers  map[string]string `json:"headers"`
}

// Count is a search result count. When Exact is false, Total is a lower bound.
type Count struct {
	Total int
	Exact bool
}

type Criteria struct {
	From           string `json:"from"`
	To             string `json:"to"`
	Subject        string `json:"subject"`
	Query          string `json:"query"`
	NegatedQuery   string `json:"negatedQuery"`  //nolint:tagliatelle // gog's output field
	HasAttachment  bool   `json:"hasAttachment"` //nolint:tagliatelle // gog's output field
	Size           int64  `json:"size"`
	SizeComparison string `json:"sizeComparison"` //nolint:tagliatelle // gog's output field
}

type Action struct {
	AddLabelIDs    []string `json:"addLabelIds"`    //nolint:tagliatelle // gog's output field
	RemoveLabelIDs []string `json:"removeLabelIds"` //nolint:tagliatelle // gog's output field
	Forward        string   `json:"forward"`
}

type Filter struct {
	ID       string   `json:"id"`
	Criteria Criteria `json:"criteria"`
	Action   Action   `json:"action"`
}

type Inbox struct {
	Total  int
	Unread int
}

type Client struct {
	Runner gog.Runner
	Email  string
}

// Search returns up to limit messages matching query, newest first. A limit of
// zero reads every page.
func (c Client) Search(
	ctx context.Context,
	query string,
	limit int,
	headers ...string,
) ([]Message, error) {
	var (
		messages []Message
		page     string
	)

	for {
		size := maxPageSize
		if limit > 0 {
			size = min(maxPageSize, limit-len(messages))
		}

		args := []string{"--json", "messages", "search", "--max=" + strconv.Itoa(size)}
		if page != "" {
			args = append(args, "--page="+page)
		}

		for _, header := range headers {
			args = append(args, "--headers="+header)
		}

		var result struct {
			Messages []gogMessage `json:"messages"`
			Next     string       `json:"nextPageToken"` //nolint:tagliatelle // gog's output field
		}

		if err := c.decode(ctx, &result, append(args, "--", query)...); err != nil {
			return nil, fmt.Errorf("search %q: %w", query, err)
		}

		for _, m := range result.Messages {
			messages = append(messages, Message(m))
		}

		page = result.Next
		if page == "" || (limit > 0 && len(messages) >= limit) {
			return messages, nil
		}
	}
}

func (c Client) Count(ctx context.Context, query string) (Count, error) {
	var result struct {
		Total   *int `json:"totalMatches"`        //nolint:tagliatelle // gog's output field
		AtLeast *int `json:"totalMatchesAtLeast"` //nolint:tagliatelle // gog's output field
	}

	err := c.decode(
		ctx,
		&result,
		"--json",
		"messages",
		"search",
		"--count",
		"--ids-only",
		"--max=1",
		"--",
		query,
	)
	if err != nil {
		return Count{}, fmt.Errorf("count %q: %w", query, err)
	}

	switch {
	case result.Total != nil:
		return Count{Total: *result.Total, Exact: true}, nil
	case result.AtLeast != nil:
		return Count{Total: *result.AtLeast, Exact: false}, nil
	default:
		return Count{Total: 0, Exact: true}, nil
	}
}

func (c Client) Filters(ctx context.Context) ([]Filter, error) {
	var result struct {
		Filters []Filter `json:"filters"`
	}

	if err := c.decode(ctx, &result, "--json", "settings", "filters", "list"); err != nil {
		return nil, fmt.Errorf("list filters: %w", err)
	}

	return result.Filters, nil
}

// LabelNames maps label IDs to their display names.
func (c Client) LabelNames(ctx context.Context) (map[string]string, error) {
	var result struct {
		Labels []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"labels"`
	}

	if err := c.decode(ctx, &result, "--json", "labels", "list"); err != nil {
		return nil, fmt.Errorf("list labels: %w", err)
	}

	names := make(map[string]string, len(result.Labels))
	for _, label := range result.Labels {
		names[label.ID] = label.Name
	}

	return names, nil
}

func (c Client) Inbox(ctx context.Context) (Inbox, error) {
	var result struct {
		Label struct {
			Total  int `json:"messagesTotal"`  //nolint:tagliatelle // gog's output field
			Unread int `json:"messagesUnread"` //nolint:tagliatelle // gog's output field
		} `json:"label"`
	}

	if err := c.decode(ctx, &result, "--json", "labels", "get", "INBOX"); err != nil {
		return Inbox{}, fmt.Errorf("read inbox counts: %w", err)
	}

	return Inbox{Total: result.Label.Total, Unread: result.Label.Unread}, nil
}

func (c Client) decode(ctx context.Context, target any, args ...string) error {
	output, err := c.Runner.Output(ctx, c.Email, gog.GroupGmail, args)
	if err != nil {
		return fmt.Errorf("run gog: %w", err)
	}

	if err := json.Unmarshal(output, target); err != nil {
		return fmt.Errorf("decode gog output: %w", err)
	}

	return nil
}
