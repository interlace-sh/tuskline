// SPDX-License-Identifier: Apache-2.0

package lint

import (
	"fmt"
	"strings"

	pgquery "github.com/pganalyze/pg_query_go/v6"
	pg_query "github.com/wasilibs/go-pgquery"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// BigTable is the size at which SET NOT NULL is treated as a long scan.
// One hundred mebibytes is large enough to matter under an exclusive lock
// and small enough to catch a table that has left the development size.
const BigTable = 100 << 20

// TableStat is live size and activity for one table. Known is false when
// the table was not looked up.
type TableStat struct {
	Bytes  int64
	Active int
	Known  bool
}

// Stats holds the live facts the rules consult. A nil map means nothing
// was collected, which the size rule treats as unknown rather than empty.
type Stats struct {
	Tables map[string]TableStat
}

// Finding is one lock or rewrite risk. It has not been applied.
type Finding struct {
	Rule      string `json:"rule"`
	Summary   string `json:"summary"`
	Statement string `json:"statement"`
}

// Check parses sql and returns lock-aware findings in statement order.
func Check(sql string, stats Stats) ([]Finding, error) {
	tree, err := pg_query.Parse(sql)
	if err != nil {
		return nil, fmt.Errorf("parse migration: %w", err)
	}
	var out []Finding
	for _, raw := range tree.GetStmts() {
		text := statementText(sql, raw)
		stmt := raw.GetStmt()
		if stmt == nil {
			continue
		}
		if alter := stmt.GetAlterTableStmt(); alter != nil {
			out = append(out, checkAlter(alter, text, stats)...)
		}
		if idx := stmt.GetIndexStmt(); idx != nil {
			out = append(out, checkIndex(idx, text)...)
		}
	}
	return out, nil
}

// Report renders findings as one block per item, for golden tests and the CLI.
func Report(findings []Finding) string {
	var b strings.Builder
	for i, f := range findings {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "%s: %s\n%s\n", f.Rule, f.Summary, f.Statement)
	}
	return b.String()
}

func checkIndex(idx *pgquery.IndexStmt, text string) []Finding {
	if idx.GetConcurrent() {
		return nil
	}
	table := qualified(idx.GetRelation())
	if table == "" {
		table = "the table"
	}
	return []Finding{{
		Rule:      "create-index",
		Summary:   fmt.Sprintf("CREATE INDEX on %s locks writes. CONCURRENTLY builds it without that lock and cannot run inside a transaction.", table),
		Statement: text,
	}}
}

func checkAlter(alter *pgquery.AlterTableStmt, text string, stats Stats) []Finding {
	schema, name := relParts(alter.GetRelation())
	table := qualified(alter.GetRelation())
	var out []Finding
	for _, node := range alter.GetCmds() {
		cmd := node.GetAlterTableCmd()
		if cmd == nil {
			continue
		}
		switch cmd.GetSubtype() {
		case pgquery.AlterTableType_AT_AddColumn:
			if f, ok := volatileDefault(cmd, table, text); ok {
				out = append(out, f)
			}
		case pgquery.AlterTableType_AT_AlterColumnType:
			col := cmd.GetName()
			if col == "" {
				col = "a column"
			}
			out = append(out, Finding{
				Rule:      "alter-type",
				Summary:   fmt.Sprintf("Changing the type of %s.%s rewrites the table.", table, col),
				Statement: text,
			})
		case pgquery.AlterTableType_AT_SetNotNull:
			out = append(out, setNotNull(cmd, schema, name, table, text, stats)...)
		}
	}
	return out
}

func volatileDefault(cmd *pgquery.AlterTableCmd, table, text string) (Finding, bool) {
	col := cmd.GetDef().GetColumnDef()
	if col == nil {
		return Finding{}, false
	}
	var expr *pgquery.Node
	for _, c := range col.GetConstraints() {
		con := c.GetConstraint()
		if con != nil && con.GetContype() == pgquery.ConstrType_CONSTR_DEFAULT {
			expr = con.GetRawExpr()
		}
	}
	if expr == nil {
		return Finding{}, false
	}
	names := functionNames(expr)
	if len(names) == 0 {
		return Finding{}, false
	}
	var volatile, unknown []string
	for _, n := range names {
		switch {
		case stableFuncs[n]:
		case volatileFuncs[n]:
			volatile = append(volatile, n)
		default:
			unknown = append(unknown, n)
		}
	}
	column := col.GetColname()
	if column == "" {
		column = "a column"
	}
	switch {
	case len(volatile) > 0:
		return Finding{
			Rule:      "volatile-default",
			Summary:   fmt.Sprintf("Adding %s with default %s() rewrites %s because the default is volatile.", column, strings.Join(volatile, "(), "), table),
			Statement: text,
		}, true
	case len(unknown) > 0:
		return Finding{
			Rule:      "volatile-default",
			Summary:   fmt.Sprintf("Adding %s calls %s. A volatile default rewrites %s; confirm the function is stable.", column, strings.Join(unknown, ", "), table),
			Statement: text,
		}, true
	default:
		return Finding{}, false
	}
}

func setNotNull(cmd *pgquery.AlterTableCmd, schema, name, table, text string, stats Stats) []Finding {
	stat, known := stats.lookup(schema, name)
	col := cmd.GetName()
	if col == "" {
		col = "a column"
	}
	if known && stat.Bytes < BigTable && stat.Active == 0 {
		return nil
	}
	var why string
	switch {
	case !known:
		why = fmt.Sprintf("the size of %s is not known", table)
	case stat.Bytes >= BigTable && stat.Active > 0:
		why = fmt.Sprintf("%s is %s and %d sessions are using it", table, sizeText(stat.Bytes), stat.Active)
	case stat.Bytes >= BigTable:
		why = fmt.Sprintf("%s is %s", table, sizeText(stat.Bytes))
	default:
		why = fmt.Sprintf("%d sessions are using %s", stat.Active, table)
	}
	return []Finding{{
		Rule:      "set-not-null",
		Summary:   fmt.Sprintf("SET NOT NULL on %s scans %s under a lock: %s.", col, table, why),
		Statement: text,
	}}
}

func (s Stats) lookup(schema, name string) (TableStat, bool) {
	if name == "" || s.Tables == nil {
		return TableStat{}, false
	}
	keys := []string{name}
	if schema != "" {
		keys = append([]string{schema + "." + name}, keys...)
	} else {
		keys = append([]string{"public." + name}, keys...)
	}
	for _, key := range keys {
		if stat, ok := s.Tables[key]; ok && stat.Known {
			return stat, true
		}
	}
	return TableStat{}, false
}

func functionNames(expr *pgquery.Node) []string {
	var names []string
	seen := map[string]bool{}
	walk(expr, func(m proto.Message) {
		call, ok := m.(*pgquery.FuncCall)
		if !ok {
			return
		}
		var parts []string
		for _, n := range call.GetFuncname() {
			if s := n.GetString_(); s != nil && s.GetSval() != "" {
				parts = append(parts, strings.ToLower(s.GetSval()))
			}
		}
		if len(parts) == 0 {
			return
		}
		name := parts[len(parts)-1]
		if seen[name] {
			return
		}
		seen[name] = true
		names = append(names, name)
	})
	return names
}

func walk(m proto.Message, fn func(proto.Message)) {
	if m == nil {
		return
	}
	ref := m.ProtoReflect()
	if !ref.IsValid() {
		return
	}
	fn(m)
	ref.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if fd.Kind() != protoreflect.MessageKind {
			return true
		}
		if fd.IsList() {
			list := v.List()
			for i := 0; i < list.Len(); i++ {
				item := list.Get(i).Message()
				if item.IsValid() {
					walk(item.Interface(), fn)
				}
			}
			return true
		}
		if fd.IsMap() {
			return true
		}
		msg := v.Message()
		if msg.IsValid() {
			walk(msg.Interface(), fn)
		}
		return true
	})
}

func statementText(sql string, raw *pgquery.RawStmt) string {
	start := int(raw.GetStmtLocation())
	length := int(raw.GetStmtLen())
	if start < 0 || length <= 0 || start+length > len(sql) {
		return strings.TrimSpace(sql)
	}
	return strings.TrimSpace(sql[start : start+length])
}

func relParts(r *pgquery.RangeVar) (string, string) {
	if r == nil {
		return "", ""
	}
	return r.GetSchemaname(), r.GetRelname()
}

func qualified(r *pgquery.RangeVar) string {
	schema, name := relParts(r)
	if name == "" {
		return ""
	}
	if schema == "" {
		return name
	}
	return schema + "." + name
}

func sizeText(n int64) string {
	const mb = 1 << 20
	if n >= mb && n%mb == 0 {
		return fmt.Sprintf("%dMB", n/mb)
	}
	return fmt.Sprintf("%d bytes", n)
}

// stableFuncs are defaults that do not rewrite the table on PostgreSQL 11+.
var stableFuncs = map[string]bool{
	"now":                   true,
	"transaction_timestamp": true,
	"statement_timestamp":   true,
	"current_timestamp":     true,
	"current_date":          true,
	"current_time":          true,
	"localtimestamp":        true,
	"localtime":             true,
}

// volatileFuncs are defaults that rewrite every row.
var volatileFuncs = map[string]bool{
	"clock_timestamp":    true,
	"random":             true,
	"gen_random_uuid":    true,
	"nextval":            true,
	"timeofday":          true,
	"uuid_generate_v1":   true,
	"uuid_generate_v4":   true,
	"uuid_generate_v1mc": true,
}
