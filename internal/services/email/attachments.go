package email

// Attachment is a single file attached to an outbound email.
type Attachment struct {
	Filename    string
	ContentType string
	Content     []byte
}
