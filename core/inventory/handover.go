// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"fmt"
	"net/url"
	"strings"
)

func (c Cluster) validateHandover() error {
	if c.Patroni != nil {
		u, err := url.Parse(c.Patroni.URL)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
			return fmt.Errorf("patroni url must be http or https and must not contain a password")
		}
	}
	if c.Backup != nil {
		b := c.Backup
		if b.Tool != "pgbackrest" || b.Stanza == "" || b.Repo != "local-volume" || b.Path == "" || strings.ContainsAny(b.Path, "\r\n") {
			return fmt.Errorf("backup must be a pgBackRest stanza on a local volume")
		}
	}
	if c.RestoreDrill != nil {
		d := c.RestoreDrill
		if d.Status != "scheduled" || d.Command != "tl backup verify" || d.Due.IsZero() {
			return fmt.Errorf("restore drill must be scheduled for tl backup verify")
		}
	}
	return nil
}
