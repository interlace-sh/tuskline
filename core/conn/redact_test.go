// SPDX-License-Identifier: Apache-2.0

package conn

import (
	"strings"
	"testing"
)

func TestRedact(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, in, want string
	}{
		{"url password", "postgres://app:s3cret@db.example.com:5432/orders", "postgres://app:***@db.example.com:5432/orders"},
		{"postgresql scheme", "postgresql://app:s3cret@db/orders?sslmode=verify-full", "postgresql://app:***@db/orders?sslmode=verify-full"},
		{"url without password", "postgres://app@db/orders", "postgres://app@db/orders"},
		{"url escaped password", "postgres://app:p%40ss%3Aw0rd@db/orders", "postgres://app:***@db/orders"},
		{"url password parameter", "postgres://db/orders?user=app&password=s3cret", "postgres://db/orders?password=***&user=app"},
		{"url sslpassword parameter", "postgres://app@db/orders?sslpassword=keypass", "postgres://app@db/orders?sslpassword=***"},
		{"unparseable url", "postgres://app:s3cret@db:notaport/orders?password=x y", "postgres://app:***@db:notaport/orders?password=***"},
		{"keyword", "host=db user=app password=s3cret dbname=orders", "host=db user=app password=*** dbname=orders"},
		{"keyword quoted", "host=db password='s3 cr\\'et' dbname=orders", "host=db password=*** dbname=orders"},
		{"keyword spaces around =", "host=db password = s3cret", "host=db password = ***"},
		{"keyword upper case", "host=db PASSWORD=s3cret", "host=db PASSWORD=***"},
		{"keyword sslpassword", "host=db sslpassword=keypass", "host=db sslpassword=***"},
		{"no secret", "host=db user=app", "host=db user=app"},
		{"empty", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := Redact(c.in)
			if got != c.want {
				t.Fatalf("Redact(%q) = %q, want %q", c.in, got, c.want)
			}
			for _, secret := range []string{"s3cret", "keypass", "p%40ss", "cr\\'et"} {
				if strings.Contains(got, secret) {
					t.Fatalf("Redact(%q) = %q still contains %q", c.in, got, secret)
				}
			}
		})
	}
}
