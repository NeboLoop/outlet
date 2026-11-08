package webhook

import (
	"fmt"
	"strings"

	"github.com/outlet-sh/outlet/internal/events"
)

// Events are the events a webhook can subscribe to: the ones the dispatcher
// delivers. "*" subscribes to all of them.
var Events = []string{
	events.TopicContactCreated,
	events.TopicContactUnsubscribed,
	events.TopicEmailSent,
	events.TopicEmailDelivered,
	events.TopicEmailBounced,
	events.TopicEmailComplained,
	events.TopicEmailOpened,
	events.TopicEmailClicked,
}

// EncodeEvents checks that every event exists and returns them as stored on
// a webhook: comma-separated.
func EncodeEvents(list []string) (string, error) {
	var out []string
	for _, e := range list {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if e != "*" && !isEvent(e) {
			return "", fmt.Errorf("unknown webhook event %q; events are %s", e, strings.Join(Events, ", "))
		}
		out = append(out, e)
	}
	if len(out) == 0 {
		return "", fmt.Errorf("at least one event is required")
	}
	return strings.Join(out, ","), nil
}

// DecodeEvents returns a webhook's stored events.
func DecodeEvents(stored string) []string {
	out := []string{}
	for _, e := range strings.Split(stored, ",") {
		if e = strings.TrimSpace(e); e != "" {
			out = append(out, e)
		}
	}
	return out
}

func isEvent(e string) bool {
	for _, known := range Events {
		if e == known {
			return true
		}
	}
	return false
}
