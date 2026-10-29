// SPDX-License-Identifier: Apache-2.0

package conn

import (
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
)

// ServerVersionNum returns the server_version_num of an open connection
// from the server_version the server reports at startup ("17.6",
// "18beta1", "16.4 (Debian 16.4-1)"), without a query. It refuses
// servers older than PostgreSQL 14.
func ServerVersionNum(c *pgx.Conn) (int, error) {
	v := c.PgConn().ParameterStatus("server_version")
	n, err := parseServerVersion(v)
	if err != nil {
		return 0, err
	}
	if err := CheckSupported(n); err != nil {
		return 0, err
	}
	return n, nil
}

func parseServerVersion(v string) (int, error) {
	digits := func(s string) (int, string) {
		i := 0
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		n, _ := strconv.Atoi(s[:i])
		return n, s[i:]
	}
	major, rest := digits(v)
	if major == 0 {
		return 0, fmt.Errorf("unrecognised server_version %q", v)
	}
	minor := 0
	if len(rest) > 0 && rest[0] == '.' {
		minor, _ = digits(rest[1:])
	}
	return major*10000 + minor, nil
}
