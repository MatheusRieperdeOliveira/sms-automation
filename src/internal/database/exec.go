package database

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrNotFound  = errors.New("database: record not found")
	ErrNoWhere   = errors.New("database: update/delete requires at least one where clause")
	ErrEmptyData = errors.New("database: data cannot be empty")
)

func (q *QueryBuilder) Get() ([]map[string]any, error) {
	if q.err != nil {
		return nil, q.err
	}

	rows, err := q.conn.db.Query(q.selectSQL(), q.args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRows(rows)
}

func (q *QueryBuilder) First() (map[string]any, error) {
	if q.err != nil {
		return nil, q.err
	}

	q.limit = 1
	rows, err := q.Get()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return rows[0], nil
}

func (q *QueryBuilder) Count() (int, error) {
	if q.err != nil {
		return 0, q.err
	}

	query := "SELECT COUNT(*) FROM " + quoteIdentifier(q.table) + q.whereSQL()

	var count int
	if err := q.conn.db.QueryRow(query, q.args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (q *QueryBuilder) Create(data map[string]any) error {
	if q.err != nil {
		return q.err
	}

	cols, vals, err := mapKeys(data)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		quoteIdentifier(q.table),
		strings.Join(quoteAll(cols), ", "),
		strings.Join(placeholders(1, len(vals)), ", "),
	)

	_, err = q.conn.db.Exec(query, vals...)
	return err
}

func (q *QueryBuilder) Update(data map[string]any) error {
	if q.err != nil {
		return q.err
	}
	if len(q.wheres) == 0 {
		return ErrNoWhere
	}

	cols, vals, err := mapKeys(data)
	if err != nil {
		return err
	}

	sets := make([]string, len(cols))
	args := make([]any, 0, len(vals)+len(q.args))
	for i, col := range cols {
		args = append(args, vals[i])
		sets[i] = fmt.Sprintf("%s = $%d", quoteIdentifier(col), i+1)
	}
	args = append(args, q.args...)

	where := shiftWherePlaceholders(q.whereSQL(), len(vals))
	query := fmt.Sprintf("UPDATE %s SET %s%s", quoteIdentifier(q.table), strings.Join(sets, ", "), where)

	_, err = q.conn.db.Exec(query, args...)
	return err
}

func (q *QueryBuilder) Delete() error {
	if q.err != nil {
		return q.err
	}
	if len(q.wheres) == 0 {
		return ErrNoWhere
	}

	query := "DELETE FROM " + quoteIdentifier(q.table) + q.whereSQL()
	_, err := q.conn.db.Exec(query, q.args...)
	return err
}

func (q *QueryBuilder) CreateOrUpdate(uniqueCols []string, data map[string]any) error {
	if q.err != nil {
		return q.err
	}
	if len(uniqueCols) == 0 {
		return fmt.Errorf("database: CreateOrUpdate requires at least one unique column")
	}

	unique := make(map[string]bool, len(uniqueCols))
	for _, col := range uniqueCols {
		if err := validateIdentifier(col); err != nil {
			return err
		}
		if _, ok := data[col]; !ok {
			return fmt.Errorf("database: unique column %q missing from data", col)
		}
		unique[col] = true
	}

	cols, vals, err := mapKeys(data)
	if err != nil {
		return err
	}

	updates := make([]string, 0, len(cols))
	for _, col := range cols {
		if unique[col] {
			continue
		}
		quoted := quoteIdentifier(col)
		updates = append(updates, quoted+" = EXCLUDED."+quoted)
	}

	conflict := strings.Join(quoteAll(uniqueCols), ", ")
	onConflict := " DO NOTHING"
	if len(updates) > 0 {
		onConflict = " DO UPDATE SET " + strings.Join(updates, ", ")
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s)%s",
		quoteIdentifier(q.table),
		strings.Join(quoteAll(cols), ", "),
		strings.Join(placeholders(1, len(vals)), ", "),
		conflict,
		onConflict,
	)

	_, err = q.conn.db.Exec(query, vals...)
	return err
}

func scanRows(rows *sql.Rows) ([]map[string]any, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var result []map[string]any
	for rows.Next() {
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}

		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}

		row := make(map[string]any, len(cols))
		for i, col := range cols {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	if result == nil {
		result = []map[string]any{}
	}
	return result, nil
}

func shiftWherePlaceholders(where string, offset int) string {
	if offset == 0 || where == "" {
		return where
	}

	var b strings.Builder
	i := 0
	for i < len(where) {
		if where[i] == '$' {
			j := i + 1
			n := 0
			for j < len(where) && where[j] >= '0' && where[j] <= '9' {
				n = n*10 + int(where[j]-'0')
				j++
			}
			if n > 0 {
				b.WriteString(fmt.Sprintf("$%d", n+offset))
				i = j
				continue
			}
		}
		b.WriteByte(where[i])
		i++
	}
	return b.String()
}
