package db

import (
	"database/sql"

	"github.com/siti-nabila/orm/internal/dialect"
)

func (db *DB) Exec(query string, args ...any) (sql.Result, error) {
	return db.conn.Exec(query, args...)
}

func (db *DB) Query(query string, args ...any) (*sql.Rows, error) {
	return db.conn.Query(query, args...)
}

func (db *DB) QueryRow(query string, args ...any) *sql.Row {
	return db.conn.QueryRow(query, args...)
}

func (db *DB) Begin() (*Tx, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return nil, err
	}
	return &Tx{
		tx:      tx,
		dialect: db.dialect,
	}, nil
}

func (db *DB) Dialect() dialect.Dialector {
	return db.dialect
}
