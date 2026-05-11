package gmail

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/tedelm/gmail-planner/internal/config"
	"github.com/tedelm/gmail-planner/internal/logging"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gmailapi "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Client wraps a Gmail API service authenticated as the signed-in user.
type Client struct {
	svc *gmailapi.Service
}

// NewClient builds an OAuth2-backed Gmail client, running a browser redirect flow
// when no valid token file exists.
func NewClient(ctx context.Context, cfg *config.GmailOAuthConfig, logger *logging.Logger) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("cfg is nil")
	}
	oauthCfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes: []string{
			gmailapi.GmailReadonlyScope,
			gmailapi.GmailSendScope,
		},
		Endpoint: google.Endpoint,
	}

	tok, err := readTokenFromFile(cfg.TokenPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("read token file: %w", err)
		}
		logger.Info("No saved token; starting OAuth flow in the browser.")
		tok, err = obtainTokenViaBrowser(ctx, oauthCfg, logger)
		if err != nil {
			return nil, fmt.Errorf("oauth: %w", err)
		}
		if err := writeTokenToFile(cfg.TokenPath, tok); err != nil {
			return nil, fmt.Errorf("save token: %w", err)
		}
		logger.Infof("Saved token to %s", cfg.TokenPath)
	}
	if tok.RefreshToken == "" {
		logger.Warn("Token has no refresh token; re-running OAuth flow.")
		tok, err = obtainTokenViaBrowser(ctx, oauthCfg, logger)
		if err != nil {
			return nil, fmt.Errorf("oauth: %w", err)
		}
		if err := writeTokenToFile(cfg.TokenPath, tok); err != nil {
			return nil, fmt.Errorf("save token: %w", err)
		}
	}

	ts := oauthCfg.TokenSource(ctx, tok)
	hc := oauth2.NewClient(ctx, ts)
	svc, err := gmailapi.NewService(ctx, option.WithHTTPClient(hc))
	if err != nil {
		return nil, fmt.Errorf("gmail service: %w", err)
	}
	return &Client{svc: svc}, nil
}

// ListInboxMessages returns up to maxResults recent messages in INBOX (newest first).
func (c *Client) ListInboxMessages(ctx context.Context, maxResults int64) ([]*gmailapi.Message, error) {
	maxResults = clampInboxMax(maxResults)
	call := c.svc.Users.Messages.List("me").
		LabelIds("INBOX").
		MaxResults(maxResults)
	resp, err := call.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("list inbox messages: %w", err)
	}
	if resp.Messages == nil {
		return []*gmailapi.Message{}, nil
	}
	return resp.Messages, nil
}

// ListInboxMessagesDetailed lists INBOX message IDs then loads each message with
// format=full to populate headline (Subject) and body (first text/plain part).
func (c *Client) ListInboxMessagesDetailed(ctx context.Context, maxResults int64) ([]InboxMessage, error) {
	return c.ListInboxMessagesDetailedWithQuery(ctx, maxResults, "")
}

// ListInboxMessagesDetailedWithQuery is like ListInboxMessagesDetailed but applies an
// optional Gmail search query when q is non-empty (see UsersMessagesListCall.Q).
func (c *Client) ListInboxMessagesDetailedWithQuery(ctx context.Context, maxResults int64, q string) ([]InboxMessage, error) {
	maxResults = clampInboxMax(maxResults)
	call := c.svc.Users.Messages.List("me").
		LabelIds("INBOX").
		MaxResults(maxResults)
	if strings.TrimSpace(q) != "" {
		call = call.Q(q)
	}
	listResp, err := call.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("list inbox messages: %w", err)
	}
	if listResp.Messages == nil {
		return []InboxMessage{}, nil
	}
	out := make([]InboxMessage, 0, len(listResp.Messages))
	for _, ref := range listResp.Messages {
		if ref.Id == "" {
			continue
		}
		full, err := c.svc.Users.Messages.Get("me", ref.Id).
			Format("full").
			Context(ctx).
			Do()
		if err != nil {
			return nil, fmt.Errorf("get message %s: %w", ref.Id, err)
		}
		out = append(out, messageToInboxSummary(full))
	}
	return out, nil
}

// SendAsEmail returns the authenticated user's Gmail address.
func (c *Client) SendAsEmail(ctx context.Context) (string, error) {
	prof, err := c.svc.Users.GetProfile("me").Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("get profile: %w", err)
	}
	if prof.EmailAddress == "" {
		return "", fmt.Errorf("profile has no email address")
	}
	return prof.EmailAddress, nil
}

// SendPlainText sends a simple UTF-8 plain-text message via Gmail (RFC 822, base64url raw).
func (c *Client) SendPlainText(ctx context.Context, from, to, subject, body string) error {
	raw, err := buildGmailSendRaw(from, to, subject, body)
	if err != nil {
		return err
	}
	_, err = c.svc.Users.Messages.Send("me", &gmailapi.Message{Raw: raw}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("gmail send: %w", err)
	}
	return nil
}

// SendHTML sends a UTF-8 multipart/alternative email with both plain-text and HTML bodies.
// Gmail will choose the best part to display.
func (c *Client) SendHTML(ctx context.Context, from, to, subject, plainBody, htmlBody string) error {
	raw, err := buildGmailSendRawHTML(from, to, subject, plainBody, htmlBody)
	if err != nil {
		return err
	}
	_, err = c.svc.Users.Messages.Send("me", &gmailapi.Message{Raw: raw}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("gmail send: %w", err)
	}
	return nil
}

// BuildGmailSendRaw builds the URL-safe base64-encoded raw MIME payload Gmail expects for users.messages.send.
func BuildGmailSendRaw(from, to, subject, body string) (string, error) {
	return buildGmailSendRaw(from, to, subject, body)
}

func buildGmailSendRaw(from, to, subject, body string) (string, error) {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	if from == "" || to == "" {
		return "", fmt.Errorf("from and to are required")
	}
	rfc822 := buildPlainTextRFC822(from, to, subject, body)
	enc := base64.RawURLEncoding.EncodeToString([]byte(rfc822))
	return enc, nil
}

func buildPlainTextRFC822(from, to, subject, body string) string {
	subject = strings.ReplaceAll(subject, "\r", "")
	subject = strings.ReplaceAll(subject, "\n", " ")
	subject = encodeRFC2047SubjectIfNeeded(subject)
	var b strings.Builder
	b.WriteString("From: ")
	b.WriteString(from)
	b.WriteString("\r\nTo: ")
	b.WriteString(to)
	b.WriteString("\r\nSubject: ")
	b.WriteString(subject)
	b.WriteString("\r\nMIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	b.WriteString(toCRLFBody(body))
	return b.String()
}

func buildGmailSendRawHTML(from, to, subject, plainBody, htmlBody string) (string, error) {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	if from == "" || to == "" {
		return "", fmt.Errorf("from and to are required")
	}
	rfc822, err := buildMultipartAlternativeRFC822(from, to, subject, plainBody, htmlBody)
	if err != nil {
		return "", err
	}
	enc := base64.RawURLEncoding.EncodeToString([]byte(rfc822))
	return enc, nil
}

func buildMultipartAlternativeRFC822(from, to, subject, plainBody, htmlBody string) (string, error) {
	subject = strings.ReplaceAll(subject, "\r", "")
	subject = strings.ReplaceAll(subject, "\n", " ")
	subject = encodeRFC2047SubjectIfNeeded(subject)
	boundary, err := randomBoundary()
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("From: ")
	b.WriteString(from)
	b.WriteString("\r\nTo: ")
	b.WriteString(to)
	b.WriteString("\r\nSubject: ")
	b.WriteString(subject)
	b.WriteString("\r\nMIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: multipart/alternative; boundary=")
	b.WriteString(boundary)
	b.WriteString("\r\n\r\n")

	// text/plain part
	b.WriteString("--")
	b.WriteString(boundary)
	b.WriteString("\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	b.WriteString(toCRLFBody(plainBody))
	b.WriteString("\r\n")

	// text/html part
	b.WriteString("--")
	b.WriteString(boundary)
	b.WriteString("\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	b.WriteString(toCRLFBody(htmlBody))
	b.WriteString("\r\n")

	// closing boundary
	b.WriteString("--")
	b.WriteString(boundary)
	b.WriteString("--\r\n")

	return b.String(), nil
}

func randomBoundary() (string, error) {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("random boundary: %w", err)
	}
	return fmt.Sprintf("gmailplanner_%x", buf[:]), nil
}

func encodeRFC2047SubjectIfNeeded(subject string) string {
	for _, r := range subject {
		if r > 127 {
			return mime.QEncoding.Encode("utf-8", subject)
		}
	}
	return subject
}

// TextToSimpleHTML converts plain text into a basic HTML document.
func TextToSimpleHTML(text string) string {
	escaped := html.EscapeString(text)
	return "<!doctype html><html><body><pre style=\"font-family:Segoe UI,Arial,sans-serif;white-space:pre-wrap\">" +
		escaped +
		"</pre></body></html>"
}

func toCRLFBody(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.ReplaceAll(s, "\n", "\r\n")
}

func clampInboxMax(n int64) int64 {
	if n < config.GmailInboxListMin {
		return config.GmailInboxListMin
	}
	if n > config.GmailInboxListMax {
		return config.GmailInboxListMax
	}
	return n
}

func messageToInboxSummary(m *gmailapi.Message) InboxMessage {
	if m == nil {
		return InboxMessage{}
	}
	msg := InboxMessage{
		ID:       m.Id,
		ThreadID: m.ThreadId,
	}
	if m.Payload != nil {
		msg.Headline = subjectFromPart(m.Payload)
		msg.Body = plainTextFromPart(m.Payload)
	}
	if msg.Body == "" && m.Snippet != "" {
		msg.Body = m.Snippet
	}
	return msg
}

func subjectFromPart(p *gmailapi.MessagePart) string {
	if p == nil {
		return ""
	}
	for _, h := range p.Headers {
		if strings.EqualFold(h.Name, "Subject") {
			var dec mime.WordDecoder
			out, err := dec.DecodeHeader(h.Value)
			if err != nil {
				return h.Value
			}
			return out
		}
	}
	for _, sub := range p.Parts {
		if s := subjectFromPart(sub); s != "" {
			return s
		}
	}
	return ""
}

func plainTextFromPart(p *gmailapi.MessagePart) string {
	if p == nil {
		return ""
	}
	if strings.HasPrefix(p.MimeType, "multipart/") {
		for _, sub := range p.Parts {
			if s := plainTextFromPart(sub); s != "" {
				return s
			}
		}
		return ""
	}
	if p.MimeType == "text/plain" && p.Body != nil && p.Body.Data != "" {
		if b, err := decodeGmailBodyData(p.Body.Data); err == nil && len(b) > 0 {
			return string(b)
		}
	}
	for _, sub := range p.Parts {
		if s := plainTextFromPart(sub); s != "" {
			return s
		}
	}
	return ""
}

func decodeGmailBodyData(data string) ([]byte, error) {
	if data == "" {
		return nil, nil
	}
	if b, err := base64.RawURLEncoding.DecodeString(data); err == nil {
		return b, nil
	}
	return base64.URLEncoding.DecodeString(data)
}

func readTokenFromFile(path string) (*oauth2.Token, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var t oauth2.Token
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, fmt.Errorf("decode token json: %w", err)
	}
	return &t, nil
}

func writeTokenToFile(path string, tok *oauth2.Token) error {
	b, err := json.Marshal(tok)
	if err != nil {
		return fmt.Errorf("encode token: %w", err)
	}
	if err := os.WriteFile(path, b, 0600); err != nil {
		return fmt.Errorf("write token file: %w", err)
	}
	return nil
}

func obtainTokenViaBrowser(ctx context.Context, oauthCfg *oauth2.Config, logger *logging.Logger) (*oauth2.Token, error) {
	flowCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	authURL := oauthCfg.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	u, err := url.Parse(oauthCfg.RedirectURL)
	if err != nil {
		return nil, fmt.Errorf("parse redirect URL: %w", err)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("redirect URL missing host: %q", oauthCfg.RedirectURL)
	}

	ln, err := net.Listen("tcp", u.Host)
	if err != nil {
		return nil, fmt.Errorf("listen %s: %w", u.Host, err)
	}
	defer ln.Close()

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if errVal := r.FormValue("error"); errVal != "" {
			desc := r.FormValue("error_description")
			errCh <- fmt.Errorf("oauth error: %s %s", errVal, desc)
			http.Error(w, "Authorization failed", http.StatusBadRequest)
			return
		}
		code := r.FormValue("code")
		if code == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = fmt.Fprintf(w, "<p>Waiting for authorization…</p>")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, "<p>Authorization successful. You can close this tab.</p>")
		codeCh <- code
	})

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		if serveErr := srv.Serve(ln); serveErr != nil && serveErr != http.ErrServerClosed {
			errCh <- fmt.Errorf("oauth callback server: %w", serveErr)
		}
	}()
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	logger.Infof("Open this URL in your browser to authorize the app:\n%s", authURL)

	select {
	case <-flowCtx.Done():
		return nil, fmt.Errorf("waiting for authorization: %w", flowCtx.Err())
	case err := <-errCh:
		return nil, err
	case code := <-codeCh:
		tok, err := oauthCfg.Exchange(ctx, code)
		if err != nil {
			return nil, fmt.Errorf("exchange code: %w", err)
		}
		return tok, nil
	}
}
