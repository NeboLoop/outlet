package webhook

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/outlet-sh/outlet/internal/config"
	"github.com/outlet-sh/outlet/internal/svc"

	"github.com/zeromicro/go-zero/rest/pathvar"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("simulated read error")
}

// sesRequest builds a webhook POST as the router delivers it: the org ID is a
// path variable of /webhooks/ses/:orgId.
func sesRequest(body io.Reader) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/webhooks/ses/org-test", body)
	return pathvar.WithVars(req, map[string]string{"orgId": "org-test"})
}

// createSESServiceContext creates a minimal ServiceContext for SES webhook testing
func createSESServiceContext() *svc.ServiceContext {
	return &svc.ServiceContext{
		Config: config.Config{},
		// DB is nil - tests should handle this by checking responses
	}
}

// TestSESHandler_SubscriptionConfirmation tests handling of SNS subscription confirmation
func TestSESHandler_SubscriptionConfirmation(t *testing.T) {
	// Stand in for SNS: record the SubscribeURL the handler follows.
	confirmationReceived := false
	defer stubConfirm(func(u string) error {
		confirmationReceived = u == testSubscribeURL
		return nil
	})()

	svcCtx := createSESServiceContext()
	handler := SESHandler(svcCtx)

	snsMsg := snsMessage{
		Type:         "SubscriptionConfirmation",
		MessageId:    "msg-123",
		TopicArn:     "arn:aws:sns:us-east-1:123456789:test-topic",
		SubscribeURL: testSubscribeURL,
		Token:        "test-token",
	}
	payload, err := signedSNS(snsMsg)
	require.NoError(t, err)

	req := sesRequest(bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, confirmationReceived, "Subscription confirmation should have been called")

	var response map[string]interface{}
	err = json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, true, response["success"])
	assert.Equal(t, "Subscription confirmed", response["message"])
}

// TestSESHandler_SubscriptionConfirmationFailure tests handling when confirmation request fails
func TestSESHandler_SubscriptionConfirmationFailure(t *testing.T) {
	// SNS answers the confirmation with an error.
	defer stubConfirm(func(string) error { return errors.New("subscription confirmation failed: Internal Server Error") })()

	svcCtx := createSESServiceContext()
	handler := SESHandler(svcCtx)

	snsMsg := snsMessage{
		Type:         "SubscriptionConfirmation",
		MessageId:    "msg-123",
		TopicArn:     "arn:aws:sns:us-east-1:123456789:test-topic",
		SubscribeURL: testSubscribeURL,
		Token:        "test-token",
	}
	payload, err := signedSNS(snsMsg)
	require.NoError(t, err)

	req := sesRequest(bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "Failed to confirm subscription")
}

// TestSESHandler_BounceNotification tests handling of bounce notifications
// Note: This test verifies payload parsing. Full integration testing with DB would
// require a test database setup similar to store_test.go
func TestSESHandler_BounceNotification(t *testing.T) {
	// Test that bounce notifications can be parsed correctly
	bounceNotif := sesNotification{
		NotificationType: "Bounce",
		Bounce: &sesBounce{
			BounceType:    "Permanent",
			BounceSubType: "General",
			BouncedRecipients: []sesBounceRecipient{
				{
					EmailAddress:   "bounced@example.com",
					Action:         "failed",
					Status:         "5.1.1",
					DiagnosticCode: "smtp; 550 5.1.1 User unknown",
				},
			},
			Timestamp:  "2024-01-15T12:00:00.000Z",
			FeedbackId: "feedback-123",
		},
		Mail: sesMailInfo{
			Timestamp:        "2024-01-15T11:59:00.000Z",
			MessageId:        "message-123",
			Source:           "sender@example.com",
			SourceArn:        "arn:aws:ses:us-east-1:123456789:identity/example.com",
			SendingAccountId: "123456789",
			Destination:      []string{"bounced@example.com"},
		},
	}

	innerJSON, err := json.Marshal(bounceNotif)
	require.NoError(t, err)

	// Verify the notification parses correctly
	var parsed sesNotification
	err = json.Unmarshal(innerJSON, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "Bounce", parsed.NotificationType)
	assert.NotNil(t, parsed.Bounce)
	assert.Equal(t, "Permanent", parsed.Bounce.BounceType)
	assert.Equal(t, "General", parsed.Bounce.BounceSubType)
	assert.Len(t, parsed.Bounce.BouncedRecipients, 1)
	assert.Equal(t, "bounced@example.com", parsed.Bounce.BouncedRecipients[0].EmailAddress)
}

// TestSESHandler_ComplaintNotification tests handling of complaint notifications
// Note: This test verifies payload parsing. Full integration testing with DB would
// require a test database setup similar to store_test.go
func TestSESHandler_ComplaintNotification(t *testing.T) {
	// Test that complaint notifications can be parsed correctly
	complaintNotif := sesNotification{
		NotificationType: "Complaint",
		Complaint: &sesComplaint{
			ComplainedRecipients: []sesComplaintRecipient{
				{
					EmailAddress: "complained@example.com",
				},
			},
			Timestamp:             "2024-01-15T12:00:00.000Z",
			FeedbackId:            "feedback-456",
			ComplaintFeedbackType: "abuse",
		},
		Mail: sesMailInfo{
			Timestamp:        "2024-01-15T11:59:00.000Z",
			MessageId:        "message-456",
			Source:           "sender@example.com",
			SourceArn:        "arn:aws:ses:us-east-1:123456789:identity/example.com",
			SendingAccountId: "123456789",
			Destination:      []string{"complained@example.com"},
		},
	}

	innerJSON, err := json.Marshal(complaintNotif)
	require.NoError(t, err)

	// Verify the notification parses correctly
	var parsed sesNotification
	err = json.Unmarshal(innerJSON, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "Complaint", parsed.NotificationType)
	assert.NotNil(t, parsed.Complaint)
	assert.Equal(t, "abuse", parsed.Complaint.ComplaintFeedbackType)
	assert.Len(t, parsed.Complaint.ComplainedRecipients, 1)
	assert.Equal(t, "complained@example.com", parsed.Complaint.ComplainedRecipients[0].EmailAddress)
}

// TestSESHandler_DeliveryNotification tests handling of delivery notifications (should be ignored)
func TestSESHandler_DeliveryNotification(t *testing.T) {
	svcCtx := createSESServiceContext()
	handler := SESHandler(svcCtx)

	deliveryNotif := sesNotification{
		NotificationType: "Delivery",
		Mail: sesMailInfo{
			Timestamp:   "2024-01-15T11:59:00.000Z",
			MessageId:   "message-789",
			Source:      "sender@example.com",
			Destination: []string{"delivered@example.com"},
		},
	}

	innerJSON, err := json.Marshal(deliveryNotif)
	require.NoError(t, err)

	snsMsg := snsMessage{
		Type:      "Notification",
		MessageId: "msg-delivery",
		TopicArn:  "arn:aws:sns:us-east-1:123456789:ses-delivery",
		Message:   string(innerJSON),
	}
	payload, err := signedSNS(snsMsg)
	require.NoError(t, err)

	req := sesRequest(bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Delivery notifications are ignored and return 200 without DB operations
	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err = json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, true, response["success"])
	assert.Equal(t, "Notification processed", response["message"])
}

// TestSESHandler_InvalidJSON tests handling of invalid JSON payload
func TestSESHandler_InvalidJSON(t *testing.T) {
	svcCtx := createSESServiceContext()
	handler := SESHandler(svcCtx)

	req := sesRequest(bytes.NewReader([]byte(`{invalid json`)))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "Invalid message format")
}

// TestSESHandler_InvalidInnerJSON tests handling of valid SNS message with invalid inner JSON
func TestSESHandler_InvalidInnerJSON(t *testing.T) {
	svcCtx := createSESServiceContext()
	handler := SESHandler(svcCtx)

	snsMsg := snsMessage{
		Type:      "Notification",
		MessageId: "msg-invalid",
		TopicArn:  "arn:aws:sns:us-east-1:123456789:ses-bounces",
		Message:   "{invalid inner json",
	}
	payload, err := signedSNS(snsMsg)
	require.NoError(t, err)

	req := sesRequest(bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "Invalid notification format")
}

// TestSESHandler_EmptyBody tests handling of empty request body
func TestSESHandler_EmptyBody(t *testing.T) {
	svcCtx := createSESServiceContext()
	handler := SESHandler(svcCtx)

	req := sesRequest(bytes.NewReader([]byte{}))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// TestSESHandler_UnknownMessageType tests handling of unknown SNS message types
func TestSESHandler_UnknownMessageType(t *testing.T) {
	svcCtx := createSESServiceContext()
	handler := SESHandler(svcCtx)

	snsMsg := snsMessage{
		Type:      "UnknownType",
		MessageId: "msg-unknown",
		TopicArn:  "arn:aws:sns:us-east-1:123456789:test",
	}
	payload, err := signedSNS(snsMsg)
	require.NoError(t, err)

	req := sesRequest(bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// SNS signs only the types it sends: anything else is refused
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// TestSESHandler_ReadBodyError tests handling of body read errors
func TestSESHandler_ReadBodyError(t *testing.T) {
	svcCtx := createSESServiceContext()
	handler := SESHandler(svcCtx)

	req := sesRequest(&errorReader{})
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "Failed to read body")
}

// TestSESHandler_MultipleBounceRecipients tests parsing of multiple bounced recipients
func TestSESHandler_MultipleBounceRecipients(t *testing.T) {
	bounceNotif := sesNotification{
		NotificationType: "Bounce",
		Bounce: &sesBounce{
			BounceType:    "Permanent",
			BounceSubType: "General",
			BouncedRecipients: []sesBounceRecipient{
				{EmailAddress: "user1@example.com", Status: "5.1.1"},
				{EmailAddress: "user2@example.com", Status: "5.1.1"},
				{EmailAddress: "user3@example.com", Status: "5.1.1"},
			},
			Timestamp:  "2024-01-15T12:00:00.000Z",
			FeedbackId: "feedback-multi",
		},
		Mail: sesMailInfo{
			MessageId:   "message-multi",
			Source:      "sender@example.com",
			Destination: []string{"user1@example.com", "user2@example.com", "user3@example.com"},
		},
	}

	innerJSON, err := json.Marshal(bounceNotif)
	require.NoError(t, err)

	// Verify the notification parses correctly
	var parsed sesNotification
	err = json.Unmarshal(innerJSON, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "Bounce", parsed.NotificationType)
	assert.Len(t, parsed.Bounce.BouncedRecipients, 3)
	assert.Equal(t, "user1@example.com", parsed.Bounce.BouncedRecipients[0].EmailAddress)
	assert.Equal(t, "user2@example.com", parsed.Bounce.BouncedRecipients[1].EmailAddress)
	assert.Equal(t, "user3@example.com", parsed.Bounce.BouncedRecipients[2].EmailAddress)
}

// TestSESHandler_BounceTypes tests parsing of different bounce types
func TestSESHandler_BounceTypes(t *testing.T) {
	bounceTypes := []struct {
		bounceType    string
		bounceSubType string
	}{
		{"Permanent", "General"},
		{"Permanent", "NoEmail"},
		{"Permanent", "Suppressed"},
		{"Permanent", "OnAccountSuppressionList"},
		{"Transient", "General"},
		{"Transient", "MailboxFull"},
		{"Transient", "MessageTooLarge"},
		{"Transient", "ContentRejected"},
		{"Transient", "AttachmentRejected"},
		{"Undetermined", ""},
	}

	for _, bt := range bounceTypes {
		t.Run(bt.bounceType+"_"+bt.bounceSubType, func(t *testing.T) {
			bounceNotif := sesNotification{
				NotificationType: "Bounce",
				Bounce: &sesBounce{
					BounceType:    bt.bounceType,
					BounceSubType: bt.bounceSubType,
					BouncedRecipients: []sesBounceRecipient{
						{EmailAddress: "test@example.com"},
					},
					Timestamp:  "2024-01-15T12:00:00.000Z",
					FeedbackId: "feedback-type",
				},
				Mail: sesMailInfo{
					MessageId: "message-type",
					Source:    "sender@example.com",
				},
			}

			innerJSON, err := json.Marshal(bounceNotif)
			require.NoError(t, err)

			// Verify the notification parses correctly
			var parsed sesNotification
			err = json.Unmarshal(innerJSON, &parsed)
			require.NoError(t, err)

			assert.Equal(t, bt.bounceType, parsed.Bounce.BounceType)
			assert.Equal(t, bt.bounceSubType, parsed.Bounce.BounceSubType)
		})
	}
}

// TestSESHandler_ComplaintTypes tests parsing of different complaint feedback types
func TestSESHandler_ComplaintTypes(t *testing.T) {
	complaintTypes := []string{
		"abuse",
		"auth-failure",
		"fraud",
		"not-spam",
		"other",
		"virus",
	}

	for _, ct := range complaintTypes {
		t.Run(ct, func(t *testing.T) {
			complaintNotif := sesNotification{
				NotificationType: "Complaint",
				Complaint: &sesComplaint{
					ComplainedRecipients: []sesComplaintRecipient{
						{EmailAddress: "test@example.com"},
					},
					Timestamp:             "2024-01-15T12:00:00.000Z",
					FeedbackId:            "feedback-complaint",
					ComplaintFeedbackType: ct,
				},
				Mail: sesMailInfo{
					MessageId: "message-complaint",
					Source:    "sender@example.com",
				},
			}

			innerJSON, err := json.Marshal(complaintNotif)
			require.NoError(t, err)

			// Verify the notification parses correctly
			var parsed sesNotification
			err = json.Unmarshal(innerJSON, &parsed)
			require.NoError(t, err)

			assert.Equal(t, ct, parsed.Complaint.ComplaintFeedbackType)
		})
	}
}

// TestSESHandler_ConcurrentRequests tests handling of concurrent webhook requests
// This tests non-DB-touching operations like delivery notifications
func TestSESHandler_ConcurrentRequests(t *testing.T) {
	svcCtx := createSESServiceContext()
	handler := SESHandler(svcCtx)

	// Use Delivery notification type which doesn't touch DB
	deliveryNotif := sesNotification{
		NotificationType: "Delivery",
		Mail:             sesMailInfo{MessageId: "msg-concurrent"},
	}
	innerJSON, _ := json.Marshal(deliveryNotif)
	snsMsg := snsMessage{Type: "Notification", Message: string(innerJSON)}
	payload, _ := signedSNS(snsMsg)

	// Run 10 concurrent requests
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			req := sesRequest(bytes.NewReader(payload))
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusOK, rr.Code)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestSNSMessageStruct tests the snsMessage struct serialization
func TestSNSMessageStruct(t *testing.T) {
	msg := snsMessage{
		Type:             "Notification",
		MessageId:        "msg-id-123",
		TopicArn:         "arn:aws:sns:us-east-1:123456789:topic",
		Subject:          "Test Subject",
		Message:          `{"key": "value"}`,
		SubscribeURL:     "https://sns.example.com/subscribe",
		Token:            "token-123",
		Timestamp:        "2024-01-15T12:00:00.000Z",
		SignatureVersion: "1",
		Signature:        "base64signature",
		SigningCertURL:   "https://sns.example.com/cert.pem",
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var parsed snsMessage
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, msg.Type, parsed.Type)
	assert.Equal(t, msg.MessageId, parsed.MessageId)
	assert.Equal(t, msg.TopicArn, parsed.TopicArn)
	assert.Equal(t, msg.Message, parsed.Message)
}

// TestSESNotificationStruct tests the sesNotification struct serialization
func TestSESNotificationStruct(t *testing.T) {
	notif := sesNotification{
		NotificationType: "Bounce",
		Bounce: &sesBounce{
			BounceType:    "Permanent",
			BounceSubType: "General",
			BouncedRecipients: []sesBounceRecipient{
				{
					EmailAddress:   "test@example.com",
					Action:         "failed",
					Status:         "5.1.1",
					DiagnosticCode: "smtp; 550 User not found",
				},
			},
			Timestamp:  "2024-01-15T12:00:00.000Z",
			FeedbackId: "feedback-123",
		},
		Mail: sesMailInfo{
			Timestamp:        "2024-01-15T11:59:00.000Z",
			MessageId:        "message-123",
			Source:           "sender@example.com",
			SourceArn:        "arn:aws:ses:us-east-1:123456789:identity/example.com",
			SendingAccountId: "123456789",
			Destination:      []string{"test@example.com"},
		},
	}

	data, err := json.Marshal(notif)
	require.NoError(t, err)

	var parsed sesNotification
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "Bounce", parsed.NotificationType)
	assert.NotNil(t, parsed.Bounce)
	assert.Equal(t, "Permanent", parsed.Bounce.BounceType)
	assert.Len(t, parsed.Bounce.BouncedRecipients, 1)
	assert.Equal(t, "test@example.com", parsed.Bounce.BouncedRecipients[0].EmailAddress)
}

// TestConfirmSNSSubscription: a SubscribeURL is followed only when it is an
// https URL on an SNS host; anything else is refused before any request.
func TestConfirmSNSSubscription(t *testing.T) {
	hit := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	for _, u := range []string{
		server.URL,
		"not-a-valid-url",
		"http://sns.us-east-1.amazonaws.com/?Action=ConfirmSubscription",
		"https://evil.example.com/?Action=ConfirmSubscription",
		"https://sns.us-east-1.amazonaws.com.evil.example/?Action=ConfirmSubscription",
		"https://sns.us-east-1.amazonaws.com:8443/?Action=ConfirmSubscription",
		"https://user@sns.us-east-1.amazonaws.com/?Action=ConfirmSubscription",
		"https://169.254.169.254/latest/meta-data/",
	} {
		err := confirmSNSSubscription(u)
		assert.Error(t, err, u)
		assert.Contains(t, err.Error(), "not an SNS host", u)
	}
	assert.False(t, hit, "no request may reach a non-SNS host")
}
