package database

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

var allowedOperators = map[string]bool{
	"=":     true,
	"!=":    true,
	"<>":    true,
	">":     true,
	">=":    true,
	"<":     true,
	"<=":    true,
	"LIKE":  true,
	"ILIKE": true,
	"IN":    true,
}

type whereClause struct {
	conjunction  string
	field        string
	operator     string
	placeholders string
}

type orderClause struct {
	column    string
	direction string
}

type QueryBuilder struct {
	conn   *Connection
	table  string
	wheres []whereClause
	args   []any
	orders []orderClause
	limit  int
	offset int
	err    error
}

func (q *QueryBuilder) Where(field, operator string, value any) *QueryBuilder {
	return q.addWhere("AND", field, operator, value)
}

func (q *QueryBuilder) OrWhere(field, operator string, value any) *QueryBuilder {
	return q.addWhere("OR", field, operator, value)
}

func (q *QueryBuilder) OrderBy(column, direction string) *QueryBuilder {
	if q.err != nil {
		return q
	}
	if err := validateIdentifier(column); err != nil {
		q.err = err
		return q
	}

	dir := strings.ToUpper(strings.TrimSpace(direction))
	if dir != "ASC" && dir != "DESC" {
		q.err = fmt.Errorf("database: invalid order direction %q", direction)
		return q
	}

	q.orders = append(q.orders, orderClause{column: column, direction: dir})
	return q
}

func (q *QueryBuilder) Limit(n int) *QueryBuilder {
	if q.err != nil {
		return q
	}
	if n < 0 {
		q.err = fmt.Errorf("database: limit must be >= 0")
		return q
	}
	q.limit = n
	return q
}

func (q *QueryBuilder) Offset(n int) *QueryBuilder {
	if q.err != nil {
		return q
	}
	if n < 0 {
		q.err = fmt.Errorf("database: offset must be >= 0")
		return q
	}
	q.offset = n
	return q
}

func (q *QueryBuilder) addWhere(conjunction, field, operator string, value any) *QueryBuilder {
	if q.err != nil {
		return q
	}
	if err := validateIdentifier(field); err != nil {
		q.err = err
		return q
	}

	op := strings.ToUpper(strings.TrimSpace(operator))
	if !allowedOperators[op] {
		q.err = fmt.Errorf("database: unsupported operator %q", operator)
		return q
	}

	clause := whereClause{
		conjunction: conjunction,
		field:       field,
		operator:    op,
	}

	if op == "IN" {
		items, err := toAnySlice(value)
		if err != nil {
			q.err = err
			return q
		}
		if len(items) == 0 {
			q.err = fmt.Errorf("database: IN operator requires a non-empty slice")
			return q
		}

		placeholders := make([]string, len(items))
		for i, item := range items {
			q.args = append(q.args, item)
			placeholders[i] = fmt.Sprintf("$%d", len(q.args))
		}
		clause.placeholders = "(" + strings.Join(placeholders, ", ") + ")"
	} else {
		q.args = append(q.args, value)
		clause.placeholders = fmt.Sprintf("$%d", len(q.args))
	}

	q.wheres = append(q.wheres, clause)
	return q
}

func (q *QueryBuilder) whereSQL() string {
	if len(q.wheres) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(" WHERE ")
	for i, w := range q.wheres {
		if i > 0 {
			b.WriteByte(' ')
			b.WriteString(w.conjunction)
			b.WriteByte(' ')
		}
		b.WriteString(quoteIdentifier(w.field))
		b.WriteByte(' ')
		b.WriteString(w.operator)
		b.WriteByte(' ')
		b.WriteString(w.placeholders)
	}
	return b.String()
}

func (q *QueryBuilder) orderSQL() string {
	if len(q.orders) == 0 {
		return ""
	}

	parts := make([]string, len(q.orders))
	for i, o := range q.orders {
		parts[i] = quoteIdentifier(o.column) + " " + o.direction
	}
	return " ORDER BY " + strings.Join(parts, ", ")
}

func (q *QueryBuilder) limitOffsetSQL() string {
	var b strings.Builder
	if q.limit > 0 {
		b.WriteString(fmt.Sprintf(" LIMIT %d", q.limit))
	}
	if q.offset > 0 {
		b.WriteString(fmt.Sprintf(" OFFSET %d", q.offset))
	}
	return b.String()
}

func (q *QueryBuilder) selectSQL() string {
	return "SELECT * FROM " + quoteIdentifier(q.table) + q.whereSQL() + q.orderSQL() + q.limitOffsetSQL()
}

func validateIdentifier(name string) error {
	if name == "" {
		return fmt.Errorf("database: identifier cannot be empty")
	}
	for i, r := range name {
		if i == 0 {
			if r != '_' && !unicode.IsLetter(r) {
				return fmt.Errorf("database: invalid identifier %q", name)
			}
			continue
		}
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return fmt.Errorf("database: invalid identifier %q", name)
		}
	}
	return nil
}

func quoteIdentifier(name string) string {
	return `"` + name + `"`
}

func toAnySlice(value any) ([]any, error) {
	rv := reflect.ValueOf(value)
	if !rv.IsValid() || rv.Kind() != reflect.Slice {
		return nil, fmt.Errorf("database: IN operator requires a slice")
	}

	n := rv.Len()
	out := make([]any, n)
	for i := 0; i < n; i++ {
		out[i] = rv.Index(i).Interface()
	}
	return out, nil
}

func mapKeys(data map[string]any) ([]string, []any, error) {
	if len(data) == 0 {
		return nil, nil, ErrEmptyData
	}

	cols := make([]string, 0, len(data))
	vals := make([]any, 0, len(data))
	for col, val := range data {
		if err := validateIdentifier(col); err != nil {
			return nil, nil, err
		}
		cols = append(cols, col)
		vals = append(vals, val)
	}
	return cols, vals, nil
}

func placeholders(start, n int) []string {
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = fmt.Sprintf("$%d", start+i)
	}
	return out
}

func quoteAll(names []string) []string {
	out := make([]string, len(names))
	for i, name := range names {
		out[i] = quoteIdentifier(name)
	}
	return out
}
