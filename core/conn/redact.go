// SPDX-License-Identifier: Apache-2.0

package conn

import (
	"net/url"
	"regexp"
	"strings"
)

const redacted = "***"

// secretKeys are libpq parameters whose values are secrets.
var secretKeys = []string{"password", "sslpassword"}

var (
	// keyword/value form: password=secret or password='s e c r e t'.
	kvSecretRE = regexp.MustCompile(`(?i)\b(password|sslpassword)(\s*=\s*)('(?:\\.|[^'\\])*'?|[^\s]*)`)
	// Fallbacks for URLs that do not parse: scheme://user:secret@host and
	// password=secret in the query.
	urlUserinfoRE = regexp.MustCompile(`^([a-zA-Z][a-zA-Z0-9+.-]*://[^:/@]*:)[^@]*(@)`)
	urlParamRE    = regexp.MustCompile(`(?i)\b(password|sslpassword)=[^&#]*`)
)

// placeholder survives URL escaping unchanged and is swapped for *** last.
const placeholder = "TUSKLINEREDACTED"

// Redact returns dsn with every password replaced by ***, for both libpq URLs
// and keyword/value strings, including password query parameters. Use it
// before a DSN reaches a log, an error, an API response or the screen.
func Redact(dsn string) string {
	if i := strings.Index(dsn, "://"); i > 0 && !strings.ContainsAny(dsn[:i], " =") {
		return redactURL(dsn)
	}
	return kvSecretRE.ReplaceAllString(dsn, "${1}${2}"+redacted)
}

func redactURL(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		// Mask the userinfo and any password parameter without parsing.
		s := urlUserinfoRE.ReplaceAllString(dsn, "${1}"+redacted+"${2}")
		return urlParamRE.ReplaceAllString(s, "${1}="+redacted)
	}
	if u.User != nil {
		if _, ok := u.User.Password(); ok {
			u.User = url.UserPassword(u.User.Username(), placeholder)
		}
	}
	q := u.Query()
	changed := false
	for key := range q {
		for _, k := range secretKeys {
			if strings.EqualFold(key, k) {
				q.Set(key, placeholder)
				changed = true
			}
		}
	}
	if changed {
		u.RawQuery = q.Encode()
	}
	return strings.ReplaceAll(u.String(), placeholder, redacted)
}
