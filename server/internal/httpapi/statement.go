// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"fmt"
	"strings"
	"unicode"
)

// oneStatement accepts a single SQL statement, including one trailing
// semicolon. A second statement is rejected so the editor cannot stack a
// write after a read.
func oneStatement(sql string) (string, error) {
	trimmed := strings.TrimSpace(sql)
	if trimmed == "" {
		return "", fmt.Errorf("empty statement")
	}
	body := strings.TrimRightFunc(trimmed, unicode.IsSpace)
	if strings.HasSuffix(body, ";") {
		body = strings.TrimRightFunc(strings.TrimSuffix(body, ";"), unicode.IsSpace)
	}
	if body == "" {
		return "", fmt.Errorf("empty statement")
	}
	if interiorSemicolon(body) {
		return "", fmt.Errorf("only one statement is accepted")
	}
	return body, nil
}

func interiorSemicolon(sql string) bool {
	inSingle, inDouble, inLine, inBlock := false, false, false, false
	for i := 0; i < len(sql); i++ {
		c := sql[i]
		switch {
		case inLine:
			if c == '\n' {
				inLine = false
			}
		case inBlock:
			if c == '*' && i+1 < len(sql) && sql[i+1] == '/' {
				inBlock = false
				i++
			}
		case inSingle:
			if c == '\'' {
				if i+1 < len(sql) && sql[i+1] == '\'' {
					i++
					continue
				}
				inSingle = false
			}
		case inDouble:
			if c == '"' {
				if i+1 < len(sql) && sql[i+1] == '"' {
					i++
					continue
				}
				inDouble = false
			}
		default:
			switch c {
			case '\'':
				inSingle = true
			case '"':
				inDouble = true
			case '-':
				if i+1 < len(sql) && sql[i+1] == '-' {
					inLine = true
					i++
				}
			case '/':
				if i+1 < len(sql) && sql[i+1] == '*' {
					inBlock = true
					i++
				}
			case ';':
				return true
			}
		}
	}
	return false
}
