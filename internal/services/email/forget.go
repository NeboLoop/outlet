package email

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sesv2types "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

// ProviderErasure is what one of a brand's sending pathways' providers did
// when asked to forget an address.
type ProviderErasure struct {
	Pathway  string `json:"pathway"`  // the pathway's name
	Provider string `json:"provider"` // Amazon SES, SMTP relay, Mailboxes
	Result   string `json:"result"`   // what happened, in plain words
	Failed   bool   `json:"failed"`
}

// What each provider keeps that can't be deleted through it, said the same
// way on every result.
const (
	sesKeepsNoLog  = " SES keeps no per-recipient sending log that its API can delete."
	smtpNoAPI      = "An SMTP relay has no API for deleting what it logged, so nothing was sent to it. Ask the relay's operator to delete its logs for this address."
	mailboxesNoAPI = "Mailbox providers have no API for this, so nothing was sent to them. Mail sent from the brand's mailboxes stays in their Sent folders until it is deleted there."
)

// ForgetRecipient asks the provider of each of the brand's sending pathways to
// delete what it keeps about an address, as far as the provider allows. For
// Amazon SES that is the account's suppression list entry
// (DeleteSuppressedDestination); an SMTP relay and mailboxes have no such
// API, which their results say. Each SES account is asked once. Outlet's own
// bounce and suppression records are not touched: they keep the address from
// being mailed again.
func (s *Service) ForgetRecipient(ctx context.Context, orgID, address string) ([]ProviderErasure, error) {
	pathways, err := s.db.ListSendingPathways(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("list sending pathways: %w", err)
	}
	var out []ProviderErasure
	asked := map[string]string{} // SES account (region + key) -> the pathway that asked it
	for _, p := range pathways {
		r := ProviderErasure{Pathway: p.Name}
		switch p.Provider {
		case ProviderMailboxes:
			r.Provider, r.Result = "Mailboxes", mailboxesNoAPI
		case ProviderSMTP:
			r.Provider, r.Result = "SMTP relay", smtpNoAPI
		case ProviderSES, ProviderPlatform:
			region, accessKey, secretKey, err := AWSCredentials(ctx, s.db, s.cryptoSvc, orgID, p.ID)
			switch {
			case errors.Is(err, sql.ErrNoRows) && p.Provider == ProviderPlatform:
				// The platform sends through its SMTP server when it has no SES keys
				r.Provider, r.Result = "SMTP relay", smtpNoAPI
			case err != nil:
				r.Provider, r.Result, r.Failed = "Amazon SES", "Couldn't read this pathway's AWS keys: "+err.Error(), true
			default:
				r.Provider = "Amazon SES"
				account := region + "/" + accessKey
				if first, ok := asked[account]; ok {
					r.Result = "The same Amazon SES account as " + first + ", which was asked already."
					break
				}
				asked[account] = p.Name
				r.Result, r.Failed = deleteSESSuppression(ctx, &SESConfig{Region: region, AccessKey: accessKey, SecretKey: secretKey}, address)
			}
		default:
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

// deleteSESSuppression removes an address from an SES account's suppression
// list, the one per-recipient record SES's API can delete.
func deleteSESSuppression(ctx context.Context, cfg *SESConfig, address string) (result string, failed bool) {
	awsCfg, err := awsConfig(ctx, cfg)
	if err != nil {
		return "Couldn't reach Amazon SES: " + err.Error(), true
	}
	_, err = sesv2.NewFromConfig(awsCfg).DeleteSuppressedDestination(ctx, &sesv2.DeleteSuppressedDestinationInput{EmailAddress: aws.String(address)})
	var notFound *sesv2types.NotFoundException
	switch {
	case errors.As(err, &notFound):
		return "The address wasn't on the Amazon SES account's suppression list." + sesKeepsNoLog, false
	case err != nil:
		return "Amazon SES refused to remove the address from the account's suppression list: " + err.Error(), true
	}
	return "Removed the address from the Amazon SES account's suppression list." + sesKeepsNoLog, false
}
