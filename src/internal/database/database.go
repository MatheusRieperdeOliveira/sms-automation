package database

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Connection struct {
	db *sql.DB
}

func New() (*Connection, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("database: DATABASE_URL is not set")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return &Connection{db: db}, nil
}

func (c *Connection) Close() error {
	return c.db.Close()
}

func (c *Connection) Ping() error {
	return c.db.Ping()
}

func (c *Connection) Table(name string) *QueryBuilder {
	q := &QueryBuilder{conn: c}
	if err := validateIdentifier(name); err != nil {
		q.err = err
		return q
	}
	q.table = name
	return q
}
