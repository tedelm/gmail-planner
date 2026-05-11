package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

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
	q := dcfg.GmailDigestQuery
	if strings.TrimSpace(qFlag) != "" {
		q = strings.TrimSpace(qFlag)
	}

	if strings.TrimSpace(q) == "" {
		logger.Infof("Fetching inbox messages for digest (n=%d)…", limit)
	} else {
		logger.Infof("Fetching inbox messages for digest (n=%d, q=%q)…", limit, q)
	}
	msgs, err := client.ListInboxMessagesDetailedWithQuery(ctx, limit, q)
	if err != nil {
		logger.Error("Failed to list inbox:", err)
		os.Exit(1)
	}
	if len(msgs) == 0 {
		logger.Info("No messages matched; nothing to summarize.")
		return
	}
	logger.Infof("Fetched %d message(s).", len(msgs))
	logger.Infof("Summarizing with OpenAI (model=%s, weeks=%d)…", dcfg.OpenAIModel, dcfg.DigestWeeks)
	sum := digest.NewSummarizer(dcfg, nil)
	out, err := sum.SummarizeFamilyWeeksHTML(ctx, msgs, dcfg, logger)
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
	logger.Infof("Sending digest email (to=%s, subject=%q)…", dcfg.DigestToEmail, subj)
	if err := client.SendHTML(ctx, from, dcfg.DigestToEmail, subj, out.Text, out.HTML); err != nil {
		logger.Error("Failed to send email:", err)
		os.Exit(1)
	}
	logger.Info("Digest email sent.")
	if printSummary {
		fmt.Println(out.Text)
	}
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
