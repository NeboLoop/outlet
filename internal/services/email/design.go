package email

import "regexp"

// contentTag marks where a design (email_designs.html_body) takes the email's
// own content.
var contentTag = regexp.MustCompile(`\{\{\s*content\s*\}\}`)

// ApplyDesign places an email's content into a design at its {{content}} tag.
// It is the one way Outlet wraps an email: campaigns with their chosen design,
// sequence emails with theirs or the built-in layout their template type
// names. Without a design the content is returned as it is; a design that
// lacks the tag gets the content before its </body>, so the content is never
// lost. Merge tags are filled afterwards, in the design as in the content.
func ApplyDesign(design, content string) string {
	if design == "" {
		return content
	}
	if !contentTag.MatchString(design) {
		return insertBeforeBodyEnd(design, content)
	}
	return contentTag.ReplaceAllLiteralString(design, content)
}
