package mailer

import (
	"net/mail"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBuildLoginCodeMessage(t *testing.T) {
	from := &mail.Address{Name: "MyGuy", Address: "no-reply@myguy.work"}
	to := &mail.Address{Address: "jane@example.com"}

	msg := string(buildLoginCodeMessage(from, to, "123456", time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)))

	headers, body, found := strings.Cut(msg, "\r\n\r\n")
	assert.True(t, found, "headers and body separated by a blank CRLF line")
	assert.Contains(t, headers, `From: "MyGuy" <no-reply@myguy.work>`)
	assert.Contains(t, headers, "To: <jane@example.com>")
	assert.Contains(t, headers, "Subject: Your MyGuy sign-in code: 123456")
	assert.Contains(t, body, "123456")
	assert.NotContains(t, strings.ReplaceAll(msg, "\r\n", ""), "\n", "only CRLF line endings")
}

func TestRecipientAcceptsOnlyABareAddress(t *testing.T) {
	to, err := recipient("kampala.fixer@example.com")
	if err != nil || to.Address != "kampala.fixer@example.com" || to.Name != "" {
		t.Fatalf("bare address rejected: %v %v", to, err)
	}

	for _, bad := range []string{
		"a@example.com\r\nBcc: victim@example.com",
		"a@example.com\nSubject: hi",
		`"Evil" <a@example.com>`,
		"Evil <a@example.com>",
		"not-an-address",
		"",
	} {
		if _, err := recipient(bad); err == nil {
			t.Errorf("expected %q to be rejected", bad)
		}
	}
}
