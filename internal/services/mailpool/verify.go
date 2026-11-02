package mailpool

import (
	"context"
	"strings"
	"sync"

	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/services/emailval"
)

// maxVerify is the most addresses one verify call takes; verifyWorkers how
// many it checks at once.
const (
	maxVerify     = 100
	verifyWorkers = 8
)

// VerifyResult is one address checked before its first email.
type VerifyResult struct {
	Email      string
	Syntax     bool
	MX         bool
	Disposable bool
	Role       bool // info@, sales@ ...: allowed, reported only
	Blocked    bool // on the pool blocklist (hard bounce or complaint)
	OK         bool
	Reason     string
}

// Verify checks syntax, MX and disposable domains with emailval, never
// probing a mail server over SMTP, plus the pool blocklist. Role addresses
// pass.
func (s *Service) Verify(ctx context.Context, emails []string) ([]VerifyResult, error) {
	if err := s.on(); err != nil {
		return nil, err
	}
	if len(emails) == 0 || len(emails) > maxVerify {
		return nil, errorx.NewBadRequestError("emails must hold 1 to 100 addresses")
	}
	out := make([]VerifyResult, len(emails))
	errs := make([]error, len(emails))
	work := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < verifyWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range work {
				out[i], errs[i] = s.verifyOne(ctx, emails[i])
			}
		}()
	}
	for i := range emails {
		work <- i
	}
	close(work)
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *Service) verifyOne(ctx context.Context, addr string) (VerifyResult, error) {
	r, err := emailval.Validate(ctx, addr, &emailval.Options{
		CheckDisposable: true,
		AllowRole:       true,
		LookupMX:        s.lookupMX,
	})
	if err != nil {
		return VerifyResult{}, err
	}
	v := VerifyResult{
		Email:      r.Normalized,
		Syntax:     r.SyntaxOK,
		MX:         r.HasMX,
		Disposable: r.Disposable,
	}
	if i := strings.LastIndex(r.Normalized, "@"); r.SyntaxOK && i > 0 {
		v.Role = emailval.IsRoleBased(r.Normalized[:i])
	}
	if len(r.Messages) > 0 {
		v.Reason = r.Messages[0]
	}
	if r.Valid() {
		blocked, err := s.store.IsPoolAddressBlocked(ctx, r.Normalized)
		if err != nil {
			return VerifyResult{}, err
		}
		v.Blocked = blocked == 1
		if v.Blocked {
			v.Reason = "on the pool blocklist (hard bounce or complaint)"
		}
	}
	v.OK = r.Valid() && !v.Blocked
	return v, nil
}
