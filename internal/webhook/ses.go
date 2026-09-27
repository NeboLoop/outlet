package webhook

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/events"
	"github.com/outlet-sh/outlet/internal/svc"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// SESWebhookRequest defines the path parameters for SES webhook
type SESWebhookRequest struct {
	OrgID string `path:"orgId"`
}

// SNS message wrapper
type snsMessage struct {
	Type             string `json:"Type"`
	MessageId        string `json:"MessageId"`
	TopicArn         string `json:"TopicArn"`
	Subject          string `json:"Subject"`
	Message          string `json:"Message"`
	SubscribeURL     string `json:"SubscribeURL"`
	Token            string `json:"Token"`
	Timestamp        string `json:"Timestamp"`
	SignatureVersion string `json:"SignatureVersion"`
	Signature        string `json:"Signature"`
	SigningCertURL   string `json:"SigningCertURL"`
}

// SES notification types
type sesNotification struct {
	NotificationType string        `json:"notificationType"`
	Bounce           *sesBounce    `json:"bounce,omitempty"`
	Complaint        *sesComplaint `json:"complaint,omitempty"`
	Delivery         *sesDelivery  `json:"delivery,omitempty"`
	Mail             sesMailInfo   `json:"mail"`
}

type sesBounce struct {
	BounceType        string               `json:"bounceType"`
	BounceSubType     string               `json:"bounceSubType"`
	BouncedRecipients []sesBounceRecipient `json:"bouncedRecipients"`
	Timestamp         string               `json:"timestamp"`
	FeedbackId        string               `json:"feedbackId"`
}

type sesBounceRecipient struct {
	EmailAddress   string `json:"emailAddress"`
	Action         string `json:"action"`
	Status         string `json:"status"`
	DiagnosticCode string `json:"diagnosticCode"`
}

type sesComplaint struct {
	ComplainedRecipients  []sesComplaintRecipient `json:"complainedRecipients"`
	Timestamp             string                  `json:"timestamp"`
	FeedbackId            string                  `json:"feedbackId"`
	ComplaintFeedbackType string                  `json:"complaintFeedbackType"`
}

type sesComplaintRecipient struct {
	EmailAddress string `json:"emailAddress"`
}

type sesDelivery struct {
	Timestamp            string   `json:"timestamp"`
	ProcessingTimeMillis int64    `json:"processingTimeMillis"`
	Recipients           []string `json:"recipients"`
	SmtpResponse         string   `json:"smtpResponse"`
	ReportingMTA         string   `json:"reportingMTA"`
}

type sesMailInfo struct {
	Timestamp        string   `json:"timestamp"`
	MessageId        string   `json:"messageId"`
	Source           string   `json:"source"`
	SourceArn        string   `json:"sourceArn"`
	SendingAccountId string   `json:"sendingAccountId"`
	Destination      []string `json:"destination"`
}

// SESHandler returns an HTTP handler for AWS SES/SNS webhooks
// This must be a raw handler (not go-zero) to access the raw body for SNS notifications
// The orgID is extracted from the URL path: /webhooks/ses/:orgId
func SESHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse only the path: httpx.Parse would consume a JSON body before
		// the raw SNS payload below is read.
		var req SESWebhookRequest
		if err := httpx.ParsePath(r, &req); err != nil {
			fmt.Printf("[SES Webhook] Failed to parse request: %v\n", err)
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		if req.OrgID == "" {
			fmt.Printf("[SES Webhook] No org ID in path: %s\n", r.URL.Path)
			http.Error(w, "Missing org ID", http.StatusBadRequest)
			return
		}

		// Read raw body for SNS notification
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // SNS messages are at most 256 KB
		if err != nil {
			fmt.Printf("[SES Webhook] Failed to read body: %v\n", err)
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		fmt.Printf("[SES Webhook] Received for org %s: %s\n", req.OrgID, string(body))

		// Parse the SNS wrapper
		var snsMsg snsMessage
		if err := json.Unmarshal(body, &snsMsg); err != nil {
			fmt.Printf("[SES Webhook] Failed to parse SNS message: %v\n", err)
			http.Error(w, "Invalid message format", http.StatusBadRequest)
			return
		}

		ctx := context.Background()

		// Trust nothing unsigned: the signature must verify against an SNS
		// signing certificate, and the topic must belong to the AWS account
		// this org sends with (anyone can point their own SNS topic here).
		if err := verifySNSMessage(&snsMsg); err != nil {
			fmt.Printf("[SES Webhook] Rejected message for org %s: %v\n", req.OrgID, err)
			http.Error(w, "Invalid signature", http.StatusForbidden)
			return
		}
		if err := checkTopicAccount(ctx, svcCtx, req.OrgID, snsMsg.TopicArn); err != nil {
			fmt.Printf("[SES Webhook] Rejected message for org %s: %v\n", req.OrgID, err)
			http.Error(w, "Unknown topic", http.StatusForbidden)
			return
		}

		// Handle SNS subscription confirmation
		if snsMsg.Type == "SubscriptionConfirmation" {
			fmt.Printf("[SES Webhook] SNS subscription confirmation received for org %s, confirming...\n", req.OrgID)
			if err := snsConfirm(snsMsg.SubscribeURL); err != nil {
				fmt.Printf("[SES Webhook] Failed to confirm subscription: %v\n", err)
				http.Error(w, "Failed to confirm subscription", http.StatusInternalServerError)
				return
			}
			fmt.Printf("[SES Webhook] SNS subscription confirmed for topic: %s\n", snsMsg.TopicArn)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success": true, "message": "Subscription confirmed"}`))
			return
		}

		// Handle notification messages
		if snsMsg.Type == "Notification" {
			// Parse the inner SES notification
			var sesNotif sesNotification
			if err := json.Unmarshal([]byte(snsMsg.Message), &sesNotif); err != nil {
				fmt.Printf("[SES Webhook] Failed to parse SES notification: %v\n", err)
				http.Error(w, "Invalid notification format", http.StatusBadRequest)
				return
			}

			switch sesNotif.NotificationType {
			case "Bounce":
				processBounce(ctx, svcCtx, req.OrgID, &sesNotif, body)
			case "Complaint":
				processComplaint(ctx, svcCtx, req.OrgID, &sesNotif, body)
			case "Delivery":
				processDelivery(ctx, svcCtx, req.OrgID, &sesNotif)
			default:
				fmt.Printf("[SES Webhook] Ignoring notification type: %s\n", sesNotif.NotificationType)
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success": true, "message": "Notification processed"}`))
			return
		}

		// Unknown message type
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true, "message": "OK"}`))
	}
}

// snsConfirm follows a subscription's SubscribeURL. A variable so tests can
// stand in for SNS.
var snsConfirm = confirmSNSSubscription

// confirmSNSSubscription confirms a subscription by fetching its
// SubscribeURL, which must be an https URL on an SNS host.
func confirmSNSSubscription(subscribeURL string) error {
	if !isSNSURL(subscribeURL) {
		return fmt.Errorf("SubscribeURL is not an SNS host: %q", subscribeURL)
	}
	resp, err := snsClient.Get(subscribeURL)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("subscription confirmation failed: %s", string(body))
	}
	return nil
}

func processBounce(ctx context.Context, svcCtx *svc.ServiceContext, orgID string, notif *sesNotification, rawBody []byte) {
	if notif.Bounce == nil {
		return
	}

	for _, recipient := range notif.Bounce.BouncedRecipients {
		fmt.Printf("[SES Webhook] Recording bounce for %s (org: %s, type: %s, subtype: %s)\n",
			recipient.EmailAddress,
			orgID,
			notif.Bounce.BounceType,
			notif.Bounce.BounceSubType)

		_, err := svcCtx.DB.CreateEmailBounce(ctx, db.CreateEmailBounceParams{
			ID:              uuid.New().String(),
			Email:           recipient.EmailAddress,
			EmailForLower:   recipient.EmailAddress,
			BounceType:      notif.Bounce.BounceType,
			BounceSubtype:   sql.NullString{String: notif.Bounce.BounceSubType, Valid: true},
			DiagnosticCode:  sql.NullString{String: recipient.DiagnosticCode, Valid: recipient.DiagnosticCode != ""},
			SourceEmail:     sql.NullString{String: notif.Mail.Source, Valid: true},
			MessageID:       sql.NullString{String: notif.Mail.MessageId, Valid: true},
			RawNotification: sql.NullString{String: string(rawBody), Valid: true},
		})
		if err != nil {
			fmt.Printf("[SES Webhook] Failed to record bounce for %s: %v\n", recipient.EmailAddress, err)
			continue
		}

		// Emit bounce event
		bounceType := "soft"
		if notif.Bounce.BounceType == "Permanent" {
			bounceType = "hard"
		}
		if svcCtx.Events != nil {
			_ = events.Emit(svcCtx.Events, events.TopicEmailBounced, events.EmailEvent{
				OrgID:      orgID,
				EmailID:    notif.Mail.MessageId,
				ContactID:  "", // contact_id not available from SES notification
				Status:     "bounced",
				BounceType: bounceType,
				Timestamp:  time.Now(),
			})
		}

		// Also block the contact in our contacts table
		if err := svcCtx.DB.BlockContactByEmail(ctx, recipient.EmailAddress); err != nil {
			fmt.Printf("[SES Webhook] Failed to block contact %s: %v\n", recipient.EmailAddress, err)
		}
	}
}

func processComplaint(ctx context.Context, svcCtx *svc.ServiceContext, orgID string, notif *sesNotification, rawBody []byte) {
	if notif.Complaint == nil {
		return
	}

	for _, recipient := range notif.Complaint.ComplainedRecipients {
		fmt.Printf("[SES Webhook] Recording complaint for %s (org: %s, type: %s)\n",
			recipient.EmailAddress,
			orgID,
			notif.Complaint.ComplaintFeedbackType)

		_, err := svcCtx.DB.CreateEmailComplaint(ctx, db.CreateEmailComplaintParams{
			ID:              uuid.New().String(),
			Email:           recipient.EmailAddress,
			EmailForLower:   recipient.EmailAddress,
			ComplaintType:   sql.NullString{String: notif.Complaint.ComplaintFeedbackType, Valid: notif.Complaint.ComplaintFeedbackType != ""},
			FeedbackID:      sql.NullString{String: notif.Complaint.FeedbackId, Valid: true},
			SourceEmail:     sql.NullString{String: notif.Mail.Source, Valid: true},
			MessageID:       sql.NullString{String: notif.Mail.MessageId, Valid: true},
			RawNotification: sql.NullString{String: string(rawBody), Valid: true},
		})
		if err != nil {
			fmt.Printf("[SES Webhook] Failed to record complaint for %s: %v\n", recipient.EmailAddress, err)
			continue
		}

		// Emit complaint event
		if svcCtx.Events != nil {
			_ = events.Emit(svcCtx.Events, events.TopicEmailComplained, events.EmailEvent{
				OrgID:     orgID,
				EmailID:   notif.Mail.MessageId,
				ContactID: "", // contact_id not available from SES notification
				Status:    "complained",
				Timestamp: time.Now(),
			})
		}

		// Also block the contact in our contacts table
		if err := svcCtx.DB.BlockContactByEmail(ctx, recipient.EmailAddress); err != nil {
			fmt.Printf("[SES Webhook] Failed to block contact %s: %v\n", recipient.EmailAddress, err)
		}
	}
}

func processDelivery(ctx context.Context, svcCtx *svc.ServiceContext, orgID string, notif *sesNotification) {
	if notif.Delivery == nil {
		return
	}

	for _, recipient := range notif.Delivery.Recipients {
		fmt.Printf("[SES Webhook] Recording delivery for %s (org: %s)\n", recipient, orgID)

		// Emit delivery event
		if svcCtx.Events != nil {
			_ = events.Emit(svcCtx.Events, events.TopicEmailDelivered, events.EmailEvent{
				OrgID:     orgID,
				EmailID:   notif.Mail.MessageId,
				ContactID: "", // contact_id not available from SES notification
				Status:    "delivered",
				Timestamp: time.Now(),
			})
		}
	}
}
