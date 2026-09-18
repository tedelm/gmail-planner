package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/tedelm/gmail-planner/internal/calendar"
	"github.com/tedelm/gmail-planner/internal/config"
	"github.com/tedelm/gmail-planner/internal/digest"
	"github.com/tedelm/gmail-planner/internal/gmail"
	"github.com/tedelm/gmail-planner/internal/logging"
)

func main() {
	logger := logging.NewLogger(nil)
	ctx := context.Background()

	if err := config.LoadDotEnv(); err != nil {
		logger.Error("Failed to load .env:", err)
		os.Exit(1)
	}

	digestMode := flag.Bool("digest", false, "fetch inbox, summarize with OpenAI, email digest to DIGEST_TO_EMAIL")
	printSummary := flag.Bool("print-summary", false, "with -digest: also print the summary text to stdout")
	gmailQuery := flag.String("q", "", "Gmail search query (overrides GMAIL_DIGEST_QUERY when non-empty)")
	limit := flag.Int64("n", config.DefaultGmailInboxListLimit, "max inbox messages to list (1–500)")
	flag.Parse()

	gmailCfg, err := config.LoadGmailOAuthFromEnv()
	if err != nil {
		logger.Error("Invalid configuration:", err)
		os.Exit(1)
	}

	client, err := gmail.NewClient(ctx, gmailCfg, logger)
	if err != nil {
		logger.Error("Failed to create Gmail client:", err)
		os.Exit(1)
	}

	if *digestMode {
		runDigest(ctx, client, logger, *limit, *gmailQuery, *printSummary)
		return
	}

	msgs, err := client.ListInboxMessagesDetailed(ctx, *limit)
	if err != nil {
		logger.Error("Failed to list inbox:", err)
		os.Exit(1)
	}

	if len(msgs) == 0 {
		logger.Info("No messages in INBOX for this query.")
		return
	}

	out, err := json.MarshalIndent(msgs, "", "  ")
	if err != nil {
		logger.Error("Failed to encode JSON:", err)
		os.Exit(1)
	}
	fmt.Println(string(out))
}

func runDigest(ctx context.Context, client *gmail.Client, logger *logging.Logger, limit int64, qFlag string, printSummary bool) {
	logger.Info("Digest mode enabled.")
	dcfg, err := config.LoadDigestConfigFromEnv()
	if err != nil {
		logger.Error("Digest configuration:", err)
		os.Exit(1)
	}
	base := config.BuildDigestBaseQuery(dcfg, qFlag)
	effectiveDigestLabels := filterNonEmptyStrings(dcfg.GmailDigestLabels)
	if len(effectiveDigestLabels) > 0 {
		logger.Infof("GMAIL_DIGEST_LABELS active (%d), one Gmail list per label, n=%d each: %s",
			len(effectiveDigestLabels), limit, strings.Join(effectiveDigestLabels, ", "))
	} else if strings.TrimSpace(base) == "" {
		logger.Infof("Fetching inbox messages for digest (n=%d)…", limit)
	} else {
		logger.Infof("Fetching inbox messages for digest (n=%d, q=%q)…", limit, base)
	}
	msgs, err := fetchDigestInboxMessages(ctx, client, base, dcfg.GmailDigestLabels, limit, logger)
	if err != nil {
		logger.Error("Failed to list inbox:", err)
		os.Exit(1)
	}
	logger.Infof("Fetched %d unique message(s) for digest.", len(msgs))

	events, err := fetchDigestCalendarEvents(ctx, client, dcfg, logger)
	if err != nil {
		logger.Error("Failed to list calendar events:", err)
		os.Exit(1)
	}

	if len(msgs) == 0 && len(events) == 0 {
		logger.Info("No messages or calendar events matched; nothing to summarize.")
		return
	}

	logger.Infof("Summarizing with OpenAI (model=%s, weeks=%d)…", dcfg.OpenAIModel, dcfg.DigestWeeks)
	sum := digest.NewSummarizer(dcfg, nil)
	out, err := sum.SummarizeFamilyWeeksHTML(ctx, msgs, events, dcfg, logger)
	if err != nil {
		logger.Error("OpenAI summarization failed:", err)
		os.Exit(1)
	}
	logger.Info("Summarization complete.")

	logger.Info("Resolving sender email address…")
	from, err := client.SendAsEmail(ctx)
	if err != nil {
		logger.Error("Failed to resolve sender address:", err)
		os.Exit(1)
	}
	logger.Infof("Sender resolved: %s", from)
	subj := digestSubject(dcfg)
	toHeader := strings.Join(dcfg.DigestToEmails, ", ")
	logger.Infof("Sending digest email (to=%s, subject=%q)…", toHeader, subj)
	if err := client.SendHTML(ctx, from, toHeader, subj, out.Text, out.HTML); err != nil {
		logger.Error("Failed to send email:", err)
		os.Exit(1)
	}
	logger.Info("Digest email sent.")
	if printSummary {
		fmt.Println(out.Text)
	}
}

func fetchDigestCalendarEvents(ctx context.Context, gmailClient *gmail.Client, dcfg *config.DigestConfig, logger *logging.Logger) ([]calendar.Event, error) {
	calID := strings.TrimSpace(dcfg.GmailCalendarID)
	if calID == "" {
		logger.Warn("GMAIL_CALENDAR_ID is unset; skipping calendar events (digest continues with email only).")
		return nil, nil
	}
	weeks := dcfg.DigestWeeks
	if weeks < 1 {
		weeks = 1
	}
	now := time.Now()
	to := now.Add(time.Duration(weeks) * 7 * 24 * time.Hour)
	logger.Infof("Fetching calendar events (calendar=%q, from=%s, to=%s)…",
		calID, now.Format(time.RFC3339), to.Format(time.RFC3339))
	calClient, err := calendar.NewClient(ctx, gmailClient.HTTPClient())
	if err != nil {
		return nil, err
	}
	events, err := calClient.ListEvents(ctx, calID, now, to)
	if err != nil {
		return nil, err
	}
	logger.Infof("Fetched %d calendar event(s).", len(events))
	return events, nil
}

// fetchDigestInboxMessages lists messages for the digest. With no digest labels, one list uses base `q` only.
// With labels, runs one list per label (base AND that label), merges by message ID, and sorts newest first.
func fetchDigestInboxMessages(ctx context.Context, client *gmail.Client, base string, labels []string, limit int64, logger *logging.Logger) ([]gmail.InboxMessage, error) {
	effective := filterNonEmptyStrings(labels)
	if len(effective) == 0 {
		return client.ListInboxMessagesDetailedWithQuery(ctx, limit, strings.TrimSpace(base))
	}
	byID := make(map[string]gmail.InboxMessage)
	for _, label := range effective {
		q := config.JoinGmailQueryParts(base, config.DigestLabelSearchTerm(label))
		batch, err := client.ListInboxMessagesDetailedWithQuery(ctx, limit, q)
		if err != nil {
			return nil, err
		}
		if logger != nil {
			logger.Infof("Label %q: fetched %d message(s) (q=%q)", label, len(batch), q)
		}
		for _, m := range batch {
			if m.ID == "" {
				continue
			}
			if prev, ok := byID[m.ID]; ok {
				prev.DigestLabel = mergeDigestLabelNames(prev.DigestLabel, label)
				byID[m.ID] = prev
				continue
			}
			m.DigestLabel = label
			byID[m.ID] = m
		}
	}
	out := make([]gmail.InboxMessage, 0, len(byID))
	for _, m := range byID {
		out = append(out, m)
	}
	sortDigestMessagesByDateDescThenID(out)
	return out, nil
}

func filterNonEmptyStrings(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if t := strings.TrimSpace(s); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func mergeDigestLabelNames(existing, add string) string {
	add = strings.TrimSpace(add)
	if add == "" {
		return strings.TrimSpace(existing)
	}
	existing = strings.TrimSpace(existing)
	if existing == "" {
		return add
	}
	for _, p := range strings.Split(existing, ", ") {
		if p == add {
			return existing
		}
	}
	return existing + ", " + add
}

func sortDigestMessagesByDateDescThenID(msgs []gmail.InboxMessage) {
	sort.Slice(msgs, func(i, j int) bool {
		di := strings.TrimSpace(msgs[i].DateLocal)
		dj := strings.TrimSpace(msgs[j].DateLocal)
		switch {
		case di == "" && dj != "":
			return false
		case di != "" && dj == "":
			return true
		case di != dj:
			return di > dj
		default:
			return msgs[i].ID > msgs[j].ID
		}
	})
}

func digestSubject(cfg *config.DigestConfig) string {
	if strings.TrimSpace(cfg.DigestSubject) != "" {
		return cfg.DigestSubject
	}
	if cfg.DigestWeeks == 1 {
		return "Familjeöversikt — kommande vecka"
	}
	return fmt.Sprintf("Familjeöversikt — kommande %d veckor", cfg.DigestWeeks)
}
