package generate

import (
	"errors"
	"strings"
	"unicode"

	"github.com/go-openapi/spec"
	log "github.com/sirupsen/logrus"
)

// Global logger for the generate package
var logger log.FieldLogger = log.New()

// When extending the logger with local fields, always use
// defer restoreLogger(logger)
// to reset the logger after the function returns
func restoreLogger(previousLogger log.FieldLogger) {
	logger = previousLogger
}

func getRefName(ref spec.Ref) (name string, err error) {
	url := ref.GetURL()
	if url == nil {
		err = errors.New("Ref doesn't have a url")
		logger.Error(err)
		return
	}

	parts := strings.Split(url.Fragment, "/")
	if len(parts) != 3 || parts[0] != "" || parts[1] != "definitions" {
		err = errors.New("Only relative definitions are supported")
		logger.WithField("fragment", url.Fragment).Error(err)
		return
	}

	name = goFormat(parts[2])
	return
}

func goFormat(name string) string {
	// split by - and _
	words := strings.FieldsFunc(name, func(r rune) bool {
		return r == '-' || r == '_'
	})

	// capitalize first letter of each word
	titleWords := make([]string, len(words))
	for i := range words {
		titleWords[i] = strings.Title(words[i])
	}

	// uppercase common initialisms
	return handleCommonInitialisms(strings.Join(titleWords, ""))
}

// these are taken from https://github.com/golang/lint/blob/master/lint.go
var commonInitialisms = map[string]bool{
	"ACL":   true,
	"API":   true,
	"ASCII": true,
	"CPU":   true,
	"CSS":   true,
	"DNS":   true,
	"EOF":   true,
	"GUID":  true,
	"HTML":  true,
	"HTTP":  true,
	"HTTPS": true,
	"ID":    true,
	"IP":    true,
	"JSON":  true,
	"LHS":   true,
	"QPS":   true,
	"RAM":   true,
	"RHS":   true,
	"RPC":   true,
	"SLA":   true,
	"SMTP":  true,
	"SQL":   true,
	"SSH":   true,
	"TCP":   true,
	"TLS":   true,
	"TTL":   true,
	"UDP":   true,
	"UI":    true,
	"UID":   true,
	"UUID":  true,
	"URI":   true,
	"URL":   true,
	"UTF8":  true,
	"VM":    true,
	"XML":   true,
	"XMPP":  true,
	"XSRF":  true,
	"XSS":   true,
}

func handleCommonInitialisms(input string) string {
	var words []string

	nextWordStart := 0
	for s := input; s != ""; s = s[nextWordStart:] {
		nextWordStart = strings.IndexFunc(s[1:], unicode.IsUpper) + 1
		if nextWordStart <= 0 {
			nextWordStart = len(s)
		}

		word := s[:nextWordStart]
		upper := strings.ToUpper(word)
		if commonInitialisms[upper] {
			word = upper
		}

		words = append(words, word)
	}

	return strings.Join(words, "")
}
