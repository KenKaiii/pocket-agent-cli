package email

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/smtp"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

// Email is LLM-friendly email output
type Email struct {
	ID      string `json:"id"`
	From    string `json:"from"`
	To      string `json:"to"`
	ReplyTo string `json:"reply_to,omitempty"`
	Subject string `json:"subject"`
	Date    string `json:"date"`
	Preview string `json:"preview,omitempty"`
	Body    string `json:"body,omitempty"`
	Unread  bool   `json:"unread"`
}

// Mailbox info
type Mailbox struct {
	Name     string `json:"name"`
	Messages uint32 `json:"messages"`
	Unread   uint32 `json:"unread"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "email",
		Aliases: []string{"mail"},
		Short:   "Email commands (IMAP/SMTP)",
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newReadCmd())
	cmd.AddCommand(newSendCmd())
	cmd.AddCommand(newReplyCmd())
	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newMailboxesCmd())

	return cmd
}

func newMailboxesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mailboxes",
		Short: "List mailboxes/folders",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := connectIMAP()
			if err != nil {
				return err
			}
			defer c.Logout()

			mailboxes := make(chan *imap.MailboxInfo, 50)
			done := make(chan error, 1)
			go func() {
				done <- c.List("", "*", mailboxes)
			}()

			var result []Mailbox
			for m := range mailboxes {
				// Get status for each mailbox
				status, err := c.Select(m.Name, true)
				if err != nil {
					continue
				}
				result = append(result, Mailbox{
					Name:     m.Name,
					Messages: status.Messages,
					Unread:   status.Unseen,
				})
			}

			if err := <-done; err != nil {
				return output.PrintError("list_failed", err.Error(), nil)
			}

			return output.Print(result)
		},
	}

	return cmd
}

func newListCmd() *cobra.Command {
	var limit int
	var mailbox string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List emails",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := connectIMAP()
			if err != nil {
				return err
			}
			defer c.Logout()

			mbox, err := c.Select(mailbox, true)
			if err != nil {
				return output.PrintError("mailbox_error", err.Error(), nil)
			}

			if mbox.Messages == 0 {
				return output.Print([]Email{})
			}

			// Get the last N messages
			from := uint32(1)
			to := mbox.Messages
			if mbox.Messages > uint32(limit) {
				from = mbox.Messages - uint32(limit) + 1
			}

			seqSet := new(imap.SeqSet)
			seqSet.AddRange(from, to)

			// Fetch envelope and flags
			items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchUid}
			messages := make(chan *imap.Message, limit)
			done := make(chan error, 1)
			go func() {
				done <- c.Fetch(seqSet, items, messages)
			}()

			var emails []Email
			for msg := range messages {
				if msg.Envelope == nil {
					continue
				}

				from := ""
				if len(msg.Envelope.From) > 0 {
					from = formatAddress(msg.Envelope.From[0])
				}

				to := ""
				if len(msg.Envelope.To) > 0 {
					to = formatAddress(msg.Envelope.To[0])
				}

				unread := true
				for _, flag := range msg.Flags {
					if flag == imap.SeenFlag {
						unread = false
						break
					}
				}

				emails = append(emails, Email{
					ID:      fmt.Sprintf("%d", msg.Uid),
					From:    from,
					To:      to,
					Subject: decodeHeader(msg.Envelope.Subject),
					Date:    formatTime(msg.Envelope.Date),
					Unread:  unread,
				})
			}

			if err := <-done; err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			// Reverse to show newest first
			sort.Slice(emails, func(i, j int) bool {
				return i > j
			})

			return output.Print(emails)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of emails")
	cmd.Flags().StringVarP(&mailbox, "mailbox", "m", "INBOX", "Mailbox/folder")

	return cmd
}

func newReadCmd() *cobra.Command {
	var mailbox string

	cmd := &cobra.Command{
		Use:   "read [uid]",
		Short: "Read an email by UID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var uid uint32
			fmt.Sscanf(args[0], "%d", &uid)

			c, err := connectIMAP()
			if err != nil {
				return err
			}
			defer c.Logout()

			_, err = c.Select(mailbox, true)
			if err != nil {
				return output.PrintError("mailbox_error", err.Error(), nil)
			}

			seqSet := new(imap.SeqSet)
			seqSet.AddNum(uid)

			// Fetch full message
			section := &imap.BodySectionName{}
			items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, section.FetchItem()}

			messages := make(chan *imap.Message, 1)
			done := make(chan error, 1)
			go func() {
				done <- c.UidFetch(seqSet, items, messages)
			}()

			msg := <-messages
			if err := <-done; err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			if msg == nil {
				return output.PrintError("not_found", "Email not found: "+args[0], nil)
			}

			// Parse body
			body := ""
			for _, literal := range msg.Body {
				if literal == nil {
					continue
				}
				bodyBytes, err := io.ReadAll(literal)
				if err == nil {
					body = extractTextBody(bodyBytes)
				}
			}

			from := ""
			if len(msg.Envelope.From) > 0 {
				from = formatAddress(msg.Envelope.From[0])
			}

			to := ""
			if len(msg.Envelope.To) > 0 {
				to = formatAddress(msg.Envelope.To[0])
			}

			unread := true
			for _, flag := range msg.Flags {
				if flag == imap.SeenFlag {
					unread = false
					break
				}
			}

			email := Email{
				ID:      fmt.Sprintf("%d", uid),
				From:    from,
				To:      to,
				Subject: decodeHeader(msg.Envelope.Subject),
				Date:    formatTime(msg.Envelope.Date),
				Body:    body,
				Unread:  unread,
			}

			return output.Print(email)
		},
	}

	cmd.Flags().StringVarP(&mailbox, "mailbox", "m", "INBOX", "Mailbox/folder")

	return cmd
}

func newSendCmd() *cobra.Command {
	var to string
	var subject string
	var cc string

	cmd := &cobra.Command{
		Use:   "send [body]",
		Short: "Send an email",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkEmailConfig(); err != nil {
				return err
			}

			emailAddr, _ := config.Get("email_address")
			password, _ := config.Get("email_password")
			smtpServer, _ := config.Get("smtp_server")
			smtpPort, _ := config.Get("smtp_port")
			if smtpPort == "" {
				smtpPort = "587"
			}

			body := args[0]

			// Build message
			var msg bytes.Buffer
			msg.WriteString(fmt.Sprintf("From: %s\r\n", emailAddr))
			msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
			if cc != "" {
				msg.WriteString(fmt.Sprintf("Cc: %s\r\n", cc))
			}
			msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
			msg.WriteString("MIME-Version: 1.0\r\n")
			msg.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
			msg.WriteString("\r\n")
			msg.WriteString(body)

			// Connect to SMTP
			addr := fmt.Sprintf("%s:%s", smtpServer, smtpPort)
			auth := smtp.PlainAuth("", emailAddr, password, smtpServer)

			recipients := []string{to}
			if cc != "" {
				for _, addr := range strings.Split(cc, ",") {
					recipients = append(recipients, strings.TrimSpace(addr))
				}
			}

			// Try TLS first (port 587), then SSL (port 465)
			var sendErr error
			if smtpPort == "587" {
				sendErr = smtp.SendMail(addr, auth, emailAddr, recipients, msg.Bytes())
			} else if smtpPort == "465" {
				// SSL connection
				tlsConfig := &tls.Config{ServerName: smtpServer}
				conn, err := tls.Dial("tcp", addr, tlsConfig)
				if err != nil {
					return output.PrintError("connect_failed", err.Error(), nil)
				}
				defer conn.Close()

				c, err := smtp.NewClient(conn, smtpServer)
				if err != nil {
					return output.PrintError("connect_failed", err.Error(), nil)
				}

				if err = c.Auth(auth); err != nil {
					return output.PrintError("auth_failed", err.Error(), nil)
				}
				if err = c.Mail(emailAddr); err != nil {
					return output.PrintError("send_failed", err.Error(), nil)
				}
				for _, rcpt := range recipients {
					if err = c.Rcpt(rcpt); err != nil {
						return output.PrintError("send_failed", err.Error(), nil)
					}
				}
				w, err := c.Data()
				if err != nil {
					return output.PrintError("send_failed", err.Error(), nil)
				}
				_, err = w.Write(msg.Bytes())
				if err != nil {
					return output.PrintError("send_failed", err.Error(), nil)
				}
				err = w.Close()
				if err != nil {
					return output.PrintError("send_failed", err.Error(), nil)
				}
				c.Quit()
			} else {
				sendErr = smtp.SendMail(addr, auth, emailAddr, recipients, msg.Bytes())
			}

			if sendErr != nil {
				return output.PrintError("send_failed", sendErr.Error(), map[string]any{
					"smtp_server": smtpServer,
					"smtp_port":   smtpPort,
					"hint":        "For Gmail, ensure 'Less secure app access' or use App Password. Check smtp_server and smtp_port settings.",
				})
			}

			return output.Print(map[string]any{
				"status":  "sent",
				"to":      to,
				"cc":      cc,
				"subject": subject,
			})
		},
	}

	cmd.Flags().StringVar(&to, "to", "", "Recipient (required)")
	cmd.Flags().StringVar(&subject, "subject", "", "Subject (required)")
	cmd.Flags().StringVar(&cc, "cc", "", "CC recipients (comma-separated)")
	cmd.MarkFlagRequired("to")
	cmd.MarkFlagRequired("subject")

	return cmd
}

func newReplyCmd() *cobra.Command {
	var mailbox string
	var replyAll bool

	cmd := &cobra.Command{
		Use:   "reply [uid] [body]",
		Short: "Reply to an email",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkEmailConfig(); err != nil {
				return err
			}

			var uid uint32
			fmt.Sscanf(args[0], "%d", &uid)
			replyBody := args[1]

			// First, fetch the original email to get reply details
			c, err := connectIMAP()
			if err != nil {
				return err
			}
			defer c.Logout()

			_, err = c.Select(mailbox, true)
			if err != nil {
				return output.PrintError("mailbox_error", err.Error(), nil)
			}

			seqSet := new(imap.SeqSet)
			seqSet.AddNum(uid)

			items := []imap.FetchItem{imap.FetchEnvelope}

			messages := make(chan *imap.Message, 1)
			done := make(chan error, 1)
			go func() {
				done <- c.UidFetch(seqSet, items, messages)
			}()

			msg := <-messages
			if err := <-done; err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			if msg == nil || msg.Envelope == nil {
				return output.PrintError("not_found", "Email not found: "+args[0], nil)
			}

			// Determine reply recipient
			var replyTo string
			if len(msg.Envelope.ReplyTo) > 0 {
				replyTo = formatEmailOnly(msg.Envelope.ReplyTo[0])
			} else if len(msg.Envelope.From) > 0 {
				replyTo = formatEmailOnly(msg.Envelope.From[0])
			} else {
				return output.PrintError("no_sender", "Cannot determine sender to reply to", nil)
			}

			// Build CC list for reply-all
			var ccList []string
			if replyAll {
				for _, addr := range msg.Envelope.To {
					email := formatEmailOnly(addr)
					if email != "" {
						ccList = append(ccList, email)
					}
				}
				for _, addr := range msg.Envelope.Cc {
					email := formatEmailOnly(addr)
					if email != "" {
						ccList = append(ccList, email)
					}
				}
			}

			// Build subject with Re: prefix
			subject := msg.Envelope.Subject
			if !strings.HasPrefix(strings.ToLower(subject), "re:") {
				subject = "Re: " + subject
			}

			// Now send the reply using SMTP
			emailAddr, _ := config.Get("email_address")
			password, _ := config.Get("email_password")
			smtpServer, _ := config.Get("smtp_server")
			smtpPort, _ := config.Get("smtp_port")
			if smtpPort == "" {
				smtpPort = "587"
			}

			// Build message
			var msgBuf bytes.Buffer
			msgBuf.WriteString(fmt.Sprintf("From: %s\r\n", emailAddr))
			msgBuf.WriteString(fmt.Sprintf("To: %s\r\n", replyTo))
			if len(ccList) > 0 {
				msgBuf.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(ccList, ", ")))
			}
			msgBuf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
			msgBuf.WriteString(fmt.Sprintf("In-Reply-To: <%s>\r\n", msg.Envelope.MessageId))
			msgBuf.WriteString("MIME-Version: 1.0\r\n")
			msgBuf.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
			msgBuf.WriteString("\r\n")
			msgBuf.WriteString(replyBody)

			// Send via SMTP
			addr := fmt.Sprintf("%s:%s", smtpServer, smtpPort)
			auth := smtp.PlainAuth("", emailAddr, password, smtpServer)

			recipients := []string{replyTo}
			recipients = append(recipients, ccList...)

			var sendErr error
			if smtpPort == "587" {
				sendErr = smtp.SendMail(addr, auth, emailAddr, recipients, msgBuf.Bytes())
			} else if smtpPort == "465" {
				tlsConfig := &tls.Config{ServerName: smtpServer}
				conn, err := tls.Dial("tcp", addr, tlsConfig)
				if err != nil {
					return output.PrintError("connect_failed", err.Error(), nil)
				}
				defer conn.Close()

				smtpClient, err := smtp.NewClient(conn, smtpServer)
				if err != nil {
					return output.PrintError("connect_failed", err.Error(), nil)
				}

				if err = smtpClient.Auth(auth); err != nil {
					return output.PrintError("auth_failed", err.Error(), nil)
				}
				if err = smtpClient.Mail(emailAddr); err != nil {
					return output.PrintError("send_failed", err.Error(), nil)
				}
				for _, rcpt := range recipients {
					if err = smtpClient.Rcpt(rcpt); err != nil {
						return output.PrintError("send_failed", err.Error(), nil)
					}
				}
				w, err := smtpClient.Data()
				if err != nil {
					return output.PrintError("send_failed", err.Error(), nil)
				}
				_, err = w.Write(msgBuf.Bytes())
				if err != nil {
					return output.PrintError("send_failed", err.Error(), nil)
				}
				w.Close()
				smtpClient.Quit()
			} else {
				sendErr = smtp.SendMail(addr, auth, emailAddr, recipients, msgBuf.Bytes())
			}

			if sendErr != nil {
				return output.PrintError("send_failed", sendErr.Error(), nil)
			}

			return output.Print(map[string]any{
				"status":       "sent",
				"reply_to":     replyTo,
				"cc":           ccList,
				"subject":      subject,
				"original_uid": uid,
			})
		},
	}

	cmd.Flags().StringVarP(&mailbox, "mailbox", "m", "INBOX", "Mailbox containing the email")
	cmd.Flags().BoolVarP(&replyAll, "all", "a", false, "Reply to all recipients")

	return cmd
}

func newSearchCmd() *cobra.Command {
	var limit int
	var mailbox string

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search emails",
		Long:  "Search emails. Query can be text to search in subject/body, or IMAP search criteria.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]

			c, err := connectIMAP()
			if err != nil {
				return err
			}
			defer c.Logout()

			_, err = c.Select(mailbox, true)
			if err != nil {
				return output.PrintError("mailbox_error", err.Error(), nil)
			}

			// Build search criteria - search in subject and body
			criteria := imap.NewSearchCriteria()
			criteria.Or = [][2]*imap.SearchCriteria{
				{
					{Header: map[string][]string{"Subject": {query}}},
					{Body: []string{query}},
				},
			}

			uids, err := c.Search(criteria)
			if err != nil {
				return output.PrintError("search_failed", err.Error(), nil)
			}

			if len(uids) == 0 {
				return output.Print([]Email{})
			}

			// Limit results
			if len(uids) > limit {
				uids = uids[len(uids)-limit:]
			}

			seqSet := new(imap.SeqSet)
			seqSet.AddNum(uids...)

			items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchUid}
			messages := make(chan *imap.Message, limit)
			done := make(chan error, 1)
			go func() {
				done <- c.Fetch(seqSet, items, messages)
			}()

			var emails []Email
			for msg := range messages {
				if msg.Envelope == nil {
					continue
				}

				from := ""
				if len(msg.Envelope.From) > 0 {
					from = formatAddress(msg.Envelope.From[0])
				}

				to := ""
				if len(msg.Envelope.To) > 0 {
					to = formatAddress(msg.Envelope.To[0])
				}

				unread := true
				for _, flag := range msg.Flags {
					if flag == imap.SeenFlag {
						unread = false
						break
					}
				}

				emails = append(emails, Email{
					ID:      fmt.Sprintf("%d", msg.Uid),
					From:    from,
					To:      to,
					Subject: decodeHeader(msg.Envelope.Subject),
					Date:    formatTime(msg.Envelope.Date),
					Unread:  unread,
				})
			}

			if err := <-done; err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			// Reverse to show newest first
			sort.Slice(emails, func(i, j int) bool {
				return i > j
			})

			return output.Print(emails)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Max results")
	cmd.Flags().StringVarP(&mailbox, "mailbox", "m", "INBOX", "Mailbox to search")

	return cmd
}

// checkEmailConfig validates all required email credentials are set
func checkEmailConfig() error {
	required := []string{"email_address", "email_password", "imap_server", "smtp_server"}
	var missing []string

	for _, key := range required {
		val, _ := config.Get(key)
		if val == "" {
			missing = append(missing, key)
		}
	}

	if len(missing) > 0 {
		return output.PrintError("setup_required", "Email not configured", map[string]any{
			"missing":   missing,
			"setup_cmd": "pocket setup show email",
			"hint":      "Run 'pocket setup show email' for setup instructions",
		})
	}
	return nil
}

func connectIMAP() (*client.Client, error) {
	if err := checkEmailConfig(); err != nil {
		return nil, err
	}

	imapServer, _ := config.Get("imap_server")
	imapPort, _ := config.Get("imap_port")
	if imapPort == "" {
		imapPort = "993"
	}
	emailAddr, _ := config.Get("email_address")
	password, _ := config.Get("email_password")

	addr := fmt.Sprintf("%s:%s", imapServer, imapPort)

	// Connect with TLS
	c, err := client.DialTLS(addr, nil)
	if err != nil {
		return nil, output.PrintError("connect_failed", err.Error(), map[string]any{
			"server": imapServer,
			"port":   imapPort,
			"hint":   "Check imap_server and imap_port settings",
		})
	}

	// Login
	if err := c.Login(emailAddr, password); err != nil {
		c.Logout()
		return nil, output.PrintError("auth_failed", "Login failed - check credentials", map[string]any{
			"email": emailAddr,
			"hint":  "For Gmail, use an App Password (not your regular password). Go to: https://myaccount.google.com/apppasswords",
		})
	}

	return c, nil
}

func formatAddress(addr *imap.Address) string {
	if addr == nil {
		return ""
	}
	if addr.PersonalName != "" {
		return fmt.Sprintf("%s <%s@%s>", decodeHeader(addr.PersonalName), addr.MailboxName, addr.HostName)
	}
	return fmt.Sprintf("%s@%s", addr.MailboxName, addr.HostName)
}

func formatEmailOnly(addr *imap.Address) string {
	if addr == nil || addr.MailboxName == "" || addr.HostName == "" {
		return ""
	}
	return fmt.Sprintf("%s@%s", addr.MailboxName, addr.HostName)
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	now := time.Now()
	diff := now.Sub(t)

	if diff < 24*time.Hour && t.Day() == now.Day() {
		return t.Format("15:04")
	}
	if diff < 7*24*time.Hour {
		return t.Format("Mon 15:04")
	}
	if t.Year() == now.Year() {
		return t.Format("Jan 2")
	}
	return t.Format("Jan 2, 2006")
}

func decodeHeader(s string) string {
	dec := new(mime.WordDecoder)
	decoded, err := dec.DecodeHeader(s)
	if err != nil {
		return s
	}
	return decoded
}

func extractTextBody(raw []byte) string {
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		// Fallback: just return raw text
		return cleanBody(string(raw))
	}

	contentType := msg.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/plain"
	}

	mediaType, params, _ := mime.ParseMediaType(contentType)

	// Simple text/plain
	if strings.HasPrefix(mediaType, "text/plain") {
		body, err := io.ReadAll(msg.Body)
		if err != nil {
			return ""
		}
		return decodeBody(body, msg.Header.Get("Content-Transfer-Encoding"))
	}

	// Multipart
	if strings.HasPrefix(mediaType, "multipart/") {
		return extractMultipart(msg.Body, params["boundary"])
	}

	// For HTML-only emails, strip tags
	if strings.HasPrefix(mediaType, "text/html") {
		body, err := io.ReadAll(msg.Body)
		if err != nil {
			return ""
		}
		decoded := decodeBody(body, msg.Header.Get("Content-Transfer-Encoding"))
		return stripHTML(decoded)
	}

	return ""
}

func extractMultipart(r io.Reader, boundary string) string {
	if boundary == "" {
		return ""
	}

	mr := multipart.NewReader(r, boundary)
	var textPart, htmlPart string

	for {
		p, err := mr.NextPart()
		if err != nil {
			break
		}

		contentType := p.Header.Get("Content-Type")
		mediaType, _, _ := mime.ParseMediaType(contentType)

		body, err := io.ReadAll(p)
		if err != nil {
			continue
		}

		decoded := decodeBody(body, p.Header.Get("Content-Transfer-Encoding"))

		if strings.HasPrefix(mediaType, "text/plain") {
			textPart = decoded
		} else if strings.HasPrefix(mediaType, "text/html") && textPart == "" {
			htmlPart = stripHTML(decoded)
		}
	}

	if textPart != "" {
		return cleanBody(textPart)
	}
	return cleanBody(htmlPart)
}

func decodeBody(body []byte, encoding string) string {
	encoding = strings.ToLower(encoding)
	switch encoding {
	case "quoted-printable":
		decoded, err := io.ReadAll(quotedprintable.NewReader(bytes.NewReader(body)))
		if err != nil {
			return string(body)
		}
		return string(decoded)
	case "base64":
		// Already handled by IMAP library usually
		return string(body)
	default:
		return string(body)
	}
}

func stripHTML(s string) string {
	// Simple HTML tag removal
	re := regexp.MustCompile(`<[^>]*>`)
	s = re.ReplaceAllString(s, "")
	// Decode common entities
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	return cleanBody(s)
}

func cleanBody(s string) string {
	// Normalize line endings and remove excessive whitespace
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	// Remove lines with only whitespace
	lines := strings.Split(s, "\n")
	var cleaned []string
	blankCount := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			blankCount++
			if blankCount <= 2 {
				cleaned = append(cleaned, "")
			}
		} else {
			blankCount = 0
			cleaned = append(cleaned, trimmed)
		}
	}

	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}
