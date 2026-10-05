// Package contacts detects contact details (phone numbers, emails, links,
// social handles) in text that must not carry them, such as public gig notes.
// The chat service masks the same patterns in messages; both are tested
// against shared/contact-filter-cases.json so they agree.
package contacts

import "regexp"

var patterns = []*regexp.Regexp{
	// Emails
	regexp.MustCompile(`(?i)[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}`),
	// Links: with a scheme, starting www., or a bare domain on a common TLD
	regexp.MustCompile(`(?i)(?:https?|ftp)://\S+|www\.\S+|\b[a-z0-9-]+(?:\.[a-z0-9-]+)*\.(?:com|net|org|ug|co|io|me|info|biz|app|dev|xyz|link|ly|africa)\b(?:/\S*)?`),
	// Phone numbers: 9 to 15 digits, optionally +, spaced by space - . ( )
	regexp.MustCompile(`\+?\(?\d(?:[\s\-.()]*\d){8,14}`),
	// Social handles
	regexp.MustCompile(`(?:^|[^A-Za-z0-9_])@[A-Za-z0-9_][A-Za-z0-9_.]+`),
}

// Contains reports whether text contains contact details.
func Contains(text string) bool {
	for _, p := range patterns {
		if p.MatchString(text) {
			return true
		}
	}
	return false
}
