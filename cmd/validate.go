package cmd

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/outlet-sh/outlet/internal/services/emailval"

	"github.com/spf13/cobra"
)

var validateFlags struct {
	in, out, column, helo, from, shard string
	smtp                               bool
	concurrency                        int
	timeout                            time.Duration
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate a CSV of email addresses",
	Long: `Validate every address in a CSV and write it back out with validation_status
(valid, invalid, catch-all, unknown, spamtrap, abuse, do_not_mail), validation_sub_status
and validation_reason columns.

Run it on a separate host with outbound port 25 open and reverse DNS matching --helo,
never on the IP that sends mail. Re-running with the same --out resumes where it stopped.
Use --shard i/n to split domains across several hosts.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runValidate(cmd.Context())
	},
}

func init() {
	f := validateCmd.Flags()
	f.StringVar(&validateFlags.in, "in", "", "input CSV with a header row (required)")
	f.StringVar(&validateFlags.out, "out", "", "output CSV (required; appended to when resuming)")
	f.StringVar(&validateFlags.column, "column", "email", "header name of the email column")
	f.BoolVar(&validateFlags.smtp, "smtp", false, "probe mailboxes over SMTP (needs outbound port 25)")
	f.StringVar(&validateFlags.helo, "helo", "", "EHLO hostname; must match this host's reverse DNS (required with --smtp)")
	f.StringVar(&validateFlags.from, "from", "", "MAIL FROM domain (defaults to --helo)")
	f.StringVar(&validateFlags.shard, "shard", "1/1", "process only this share of domains, e.g. 2/3")
	f.IntVar(&validateFlags.concurrency, "concurrency", 20, "domains checked in parallel")
	f.DurationVar(&validateFlags.timeout, "timeout", 30*time.Second, "per-address timeout")
	_ = validateCmd.MarkFlagRequired("in")
	_ = validateCmd.MarkFlagRequired("out")
	rootCmd.AddCommand(validateCmd)
}

func runValidate(ctx context.Context) error {
	vf := validateFlags
	if vf.smtp && vf.helo == "" {
		return errors.New("--helo is required with --smtp")
	}
	var shardN, shardI uint32
	if _, err := fmt.Sscanf(vf.shard, "%d/%d", &shardI, &shardN); err != nil || shardN == 0 || shardI < 1 || shardI > shardN {
		return fmt.Errorf("invalid --shard %q, want i/n", vf.shard)
	}

	header, rows, err := readCSV(vf.in)
	if err != nil {
		return err
	}
	col := -1
	for i, h := range header {
		if strings.EqualFold(strings.TrimSpace(h), vf.column) {
			col = i
		}
	}
	if col < 0 {
		return fmt.Errorf("column %q not found in %s", vf.column, vf.in)
	}

	done, err := doneEmails(vf.out, col)
	if err != nil {
		return err
	}

	// Group by domain so each domain is probed by one worker, which keeps the
	// per-domain rate limiter from timing out parallel probes.
	byDomain := map[string][][]string{}
	for _, row := range rows {
		if col >= len(row) {
			continue
		}
		email := strings.ToLower(strings.TrimSpace(row[col]))
		if done[email] {
			continue
		}
		domain := email[strings.LastIndex(email, "@")+1:]
		h := fnv.New32a()
		h.Write([]byte(domain))
		if h.Sum32()%shardN != shardI-1 {
			continue
		}
		byDomain[domain] = append(byDomain[domain], row)
	}

	outFile, err := os.OpenFile(vf.out, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer outFile.Close()
	w := csv.NewWriter(outFile)
	if len(done) == 0 {
		if st, _ := outFile.Stat(); st.Size() == 0 {
			_ = w.Write(append(append([]string{}, header...), "validation_status", "validation_sub_status", "validation_reason"))
		}
	}

	opts := &emailval.Options{
		Timeout:         vf.timeout,
		CheckSMTP:       vf.smtp,
		CheckDisposable: true,
		SMTP:            &emailval.SMTPConfig{HeloHostname: vf.helo, FromDomain: vf.from, MaxRetries: 2},
	}

	total := 0
	for _, r := range byDomain {
		total += len(r)
	}
	fmt.Fprintf(os.Stderr, "validating %d addresses across %d domains (%d already done)\n", total, len(byDomain), len(done))

	var (
		mu       sync.Mutex
		finished int
		counts   = map[string]int{}
	)
	write := func(row []string, status, subStatus, reason string) {
		mu.Lock()
		defer mu.Unlock()
		_ = w.Write(append(append([]string{}, row...), status, subStatus, reason))
		w.Flush()
		counts[status]++
		finished++
		if finished%500 == 0 {
			fmt.Fprintf(os.Stderr, "%d/%d %v\n", finished, total, counts)
		}
	}

	domains := make(chan string)
	var wg sync.WaitGroup
	for range max(vf.concurrency, 1) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for d := range domains {
				validateDomain(ctx, d, byDomain[d], col, opts, write)
			}
		}()
	}
	for d := range byDomain {
		domains <- d
	}
	close(domains)
	wg.Wait()

	fmt.Fprintf(os.Stderr, "done: %v\n", counts)
	return w.Error()
}

func validateDomain(ctx context.Context, domain string, rows [][]string, col int, opts *emailval.Options, write func([]string, string, string, string)) {
	for _, row := range rows {
		res, _ := emailval.Validate(ctx, row[col], opts)
		write(row, res.Status, res.SubStatus, strings.Join(res.Messages, "; "))
	}
}

func readCSV(path string) ([]string, [][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	header, err := r.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("reading header of %s: %w", path, err)
	}
	rows, err := r.ReadAll()
	return header, rows, err
}

// doneEmails returns addresses already written to out, so a re-run resumes.
// ponytail: unknowns are kept as done; re-run them by feeding the unknown rows back in as --in.
func doneEmails(out string, col int) (map[string]bool, error) {
	done := map[string]bool{}
	_, rows, err := readCSV(out)
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, io.EOF) {
		return done, nil
	}
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if col < len(row) {
			done[strings.ToLower(strings.TrimSpace(row[col]))] = true
		}
	}
	return done, nil
}
