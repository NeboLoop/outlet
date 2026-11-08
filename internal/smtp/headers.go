package smtp

import (
	"net/mail"
	"strings"
)

// Custom headers for Outlet.sh SMTP ingress
const (
	HeaderOutletList     = "X-Outlet-List"     // Marketing: the slug of the list the recipient subscribed to
	HeaderOutletTags     = "X-Outlet-Tags"     // Comma-separated tags
	HeaderOutletTemplate = "X-Outlet-Template" // Template slug to use
	HeaderOutletType     = "X-Outlet-Type"     // "marketing" or "transactional" (default: transactional)
	HeaderOutletTrack    = "X-Outlet-Track"    // "opens,clicks" or "none" (default: opens,clicks)
	HeaderOutletMetaPrefix = "X-Outlet-Meta-"  // Prefix for custom metadata
)

// OutletHeaders contains parsed custom headers from an email
type OutletHeaders struct {
	ListSlug     string
	Tags         []string
	TemplateSlug string
	Type         string // "marketing" or "transactional"
	TrackOpens   bool
	TrackClicks  bool
	Meta         map[string]string
}

// ParseOutletHeaders extracts X-Outlet-* headers from an email message
func ParseOutletHeaders(msg *mail.Message) *OutletHeaders {
	h := &OutletHeaders{
		Type:        "transactional", // default
		TrackOpens:  true,            // default
		TrackClicks: true,            // default
		Meta:        make(map[string]string),
	}

	// X-Outlet-List
	if val := msg.Header.Get(HeaderOutletList); val != "" {
		h.ListSlug = strings.TrimSpace(val)
	}

	// X-Outlet-Tags (comma-separated)
	if val := msg.Header.Get(HeaderOutletTags); val != "" {
		tags := strings.Split(val, ",")
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				h.Tags = append(h.Tags, tag)
			}
		}
	}

	// X-Outlet-Template
	if val := msg.Header.Get(HeaderOutletTemplate); val != "" {
		h.TemplateSlug = strings.TrimSpace(val)
	}

	// X-Outlet-Type
	if val := msg.Header.Get(HeaderOutletType); val != "" {
		val = strings.ToLower(strings.TrimSpace(val))
		if val == "marketing" || val == "transactional" {
			h.Type = val
		}
	}

	// X-Outlet-Track
	if val := msg.Header.Get(HeaderOutletTrack); val != "" {
		val = strings.ToLower(strings.TrimSpace(val))
		if val == "none" {
			h.TrackOpens = false
			h.TrackClicks = false
		} else {
			// Parse comma-separated tracking options
			parts := strings.Split(val, ",")
			h.TrackOpens = false
			h.TrackClicks = false
			for _, part := range parts {
				part = strings.TrimSpace(part)
				switch part {
				case "opens":
					h.TrackOpens = true
				case "clicks":
					h.TrackClicks = true
				}
			}
		}
	}

	// X-Outlet-Meta-* (custom metadata)
	// Header keys are canonicalized by Go's mail package, so we need case-insensitive prefix matching
	metaPrefix := strings.ToLower(HeaderOutletMetaPrefix)
	for key := range msg.Header {
		keyLower := strings.ToLower(key)
		if strings.HasPrefix(keyLower, metaPrefix) {
			metaKey := key[len(HeaderOutletMetaPrefix):]
			if metaKey != "" {
				h.Meta[metaKey] = msg.Header.Get(key)
			}
		}
	}

	return h
}
