package campaigns

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendCampaignTestLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Send a test of a campaign to one address, built as a real send is (design, merge tags, unsubscribe link)
func NewSendCampaignTestLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendCampaignTestLogic {
	return &SendCampaignTestLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// testTrackingToken stands in for a recipient's tracking token, so the test
// shows the unsubscribe link and footer a subscriber gets. It matches no send.
const testTrackingToken = "test"

// SendCampaignTest sends the campaign as it is in the editor to one address,
// as the signed-in member of the brand. It is built by CampaignMessage and
// sent by Send, the same code as every campaign send, with sample subscriber
// details in the merge tags.
func (l *SendCampaignTestLogic) SendCampaignTest(req *types.SendCampaignTestRequest) (resp *types.Response, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}
	to := strings.TrimSpace(req.To)
	if addr, err := mail.ParseAddress(to); err != nil || addr.Address != to {
		return nil, errorx.NewBadRequestError("Enter a valid test address.")
	}
	if strings.TrimSpace(req.Subject) == "" || strings.TrimSpace(req.HtmlBody) == "" {
		return nil, errorx.NewBadRequestError("A test needs a subject and content.")
	}
	design, err := brandDesign(l.ctx, l.svcCtx.DB, orgID, req.DesignId)
	if err != nil {
		return nil, err
	}
	c := email.CampaignEmail{
		OrgID:         orgID,
		FromName:      req.FromName,
		FromEmail:     req.FromEmail,
		ReplyTo:       req.ReplyTo,
		To:            to,
		Name:          "Test Subscriber",
		Subject:       "[Test] " + req.Subject,
		HTML:          req.HtmlBody,
		Text:          req.PlainText,
		TrackingToken: testTrackingToken,
	}
	if design != nil {
		c.DesignHTML, c.DesignText = design.HtmlBody, design.PlainText.String
	}

	if err := l.svcCtx.EmailService.Send(l.ctx, l.svcCtx.EmailService.CampaignMessage(c)); err != nil {
		l.Errorf("Campaign test to %s failed: %v", to, err)
		switch {
		case errors.Is(err, email.ErrSuppressed):
			return nil, errorx.NewBadRequestError("Not sent: that address bounced, complained, or is on the suppression list.")
		case errors.Is(err, email.ErrInvalidMessage):
			return nil, errorx.NewBadRequestError(err.Error())
		}
		return nil, errorx.NewBadRequestError("The test couldn't be sent: " + err.Error())
	}
	return &types.Response{Success: true, Message: "Test sent to " + to}, nil
}
