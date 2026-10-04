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
