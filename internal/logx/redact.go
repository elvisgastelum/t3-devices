package logx

import (
	"regexp"
	"strings"
)

// Redacted replaces every secret the redactor finds.
const Redacted = "[REDACTED]"

// sensitiveKeys are field names whose values are always masked, compared
// after lowercasing and removing '_' and '-' (so pairing_code, pairingCode
// and pairing-code all match).
var sensitiveKeys = map[string]bool{
	"token":        true,
	"code":         true,
	"pairingcode":  true,
	"paircode":     true,
	"pairtoken":    true,
	"pairingtoken": true,
	"secret":       true,
	"password":     true,
	"authkey":      true,
	"credential":   true,
}

// linkChars matches URL characters up to whitespace, a quote, '<', '>' or a
// backslash, letting JSON \uXXXX escapes through.
const linkChars = `(?:[^\s"'<>\\]|\\u[0-9a-fA-F]{4})*`

// keyAlt is the regexp alternation of sensitive key spellings found in text.
const keyAlt = `token|code|pairing[_-]?code|pair[_-]?code|pair[_-]?token|pairing[_-]?token|secret|password|auth[_-]?key|credential`

var (
	// pairLinkRe matches a pair link with or without scheme and host, such as
	// https://host.example/pair#token=abc or /pair#token=abc, also when
	// percent-encoded (%2Fpair%23) or JSON-escaped (\u0026), through to the
	// next whitespace, quote or backslash that is not a \uXXXX escape.
	pairLinkRe = regexp.MustCompile(`(?i)(?:[a-z][a-z0-9+.\-]*://)?` + linkChars + `(?:/|%2F)pair(?:#|%23)` + linkChars)

	// fragTokenRe matches a token carried in any URL fragment or query, also
	// when the separator is percent-encoded or JSON-escaped.
	fragTokenRe = regexp.MustCompile(`(?i)((?:[#?&]|%23|%26|%3F|\\u0026)token(?:=|%3D))(?:[^\s"'<>&\\]|\\u[0-9a-f]{4})+`)

	// jsonStrRe matches a JSON string field with a sensitive key, also when the
	// JSON is itself escaped one or more times inside quoted strings
	// (\"token\":\"...\" or \\\"token\\\":...). The value ends at the first
	// quote, escaped or not.
	jsonStrRe = regexp.MustCompile(`(?i)(\\*"(?:` + keyAlt + `)\\*"\s*:\s*\\*")(?:[^"\\]|\\+[^"\\])*`)

	// jsonNumRe matches a JSON number or literal field with a sensitive key.
	jsonNumRe = regexp.MustCompile(`(?i)(\\*"(?:` + keyAlt + `)\\*"\s*:\s*)([0-9A-Za-z.+\-]+)`)

	// kvRe matches key=value and key: value pairs (logfmt, slog text output,
	// query strings, Go %+v struct output) with a sensitive key.
	kvRe = regexp.MustCompile(`(?i)\b((?:` + keyAlt + `)(?:=|:[ \t]*))(\[REDACTED\]|"(?:[^"\\]|\\.)*"|[^\s,;&"}\]]+)`)

	// wordRe finds candidate bare codes: runs of letters, digits, '_' and '-'.
	wordRe = regexp.MustCompile(`[A-Za-z0-9_\-]+`)

	// groupedCodeRe is the shape of a grouped pairing code: two or more groups
	// of 4 to 8 uppercase letters or digits joined by hyphens.
	groupedCodeRe = regexp.MustCompile(`^[A-Z0-9]{4,8}(?:-[A-Z0-9]{4,8})+$`)
)

// minTokenLen is the shortest run treated as a long opaque token.
const minTokenLen = 32

// Redact masks pairing secrets in s:
//
//   - pair links: any URL or path containing "/pair#", whole link masked;
//   - URL fragments or queries carrying token=...;
//   - JSON fields and key=value or key: value pairs whose key is a sensitive name
//     (token, code, pairing_code, pair_code, pair_token, secret, password,
//     auth_key, credential, in any case and with '_' or '-' variants);
//   - bare pairing codes (see isBareCode).
func Redact(s string) string {
	if s == "" {
		return s
	}
	s = pairLinkRe.ReplaceAllString(s, Redacted)
	s = fragTokenRe.ReplaceAllString(s, "${1}"+Redacted)
	s = jsonStrRe.ReplaceAllString(s, "${1}"+Redacted)
	s = jsonNumRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := jsonNumRe.FindStringSubmatch(m)
		if strings.EqualFold(sub[2], "null") {
			return m
		}
		return sub[1] + Redacted
	})
	s = kvRe.ReplaceAllString(s, "${1}"+Redacted)
	s = wordRe.ReplaceAllStringFunc(s, func(w string) string {
		if isBareCode(w) {
			return Redacted
		}
		return w
	})
	return s
}

// isBareCode reports whether w (a maximal run of letters, digits, '_' and
// '-') looks like a pairing code shown outside a link or JSON field. The rule:
//
//   - grouped code: two or more hyphen-joined groups of 4 to 8 uppercase
//     letters or digits, containing at least one letter and one digit
//     (for example ABCD-1234 or AB12-CD34-EF56); or
//   - long opaque token: at least 32 characters containing at least one
//     letter and one digit, unless it is only hex digits and hyphens
//     (UUIDs and hashes are kept for correlation).
//
// Ordinary words, dates (2026-11-24), short identifiers (UTF-8, SHA-256) and
// lowercase hyphenated names never match.
func isBareCode(w string) bool {
	letter, digit := false, false
	for _, r := range w {
		switch {
		case r >= '0' && r <= '9':
			digit = true
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
			letter = true
		}
	}
	if !letter || !digit {
		return false
	}
	if groupedCodeRe.MatchString(w) {
		return true
	}
	return len(w) >= minTokenLen && !isHexID(w)
}

// isHexID reports whether w holds only hex digits and hyphens, the shape of
// UUIDs and hashes, which stay readable so log lines can be correlated.
func isHexID(w string) bool {
	for _, r := range w {
		if !(r == '-' || (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

// isSensitiveKey reports whether a field named key must have its value masked.
func isSensitiveKey(key string) bool {
	k := strings.ToLower(key)
	k = strings.NewReplacer("_", "", "-", "").Replace(k)
	return sensitiveKeys[k]
}
