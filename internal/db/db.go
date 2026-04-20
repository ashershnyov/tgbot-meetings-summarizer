package db

import (
	"context"
	"database/sql"
	"errors"
)

// ErrNoTransactionInCtx is used when trying to commit the transaction
// passing the context with no transaction inside.
var ErrNoTransactionInCtx = errors.New("no transaction found in context")

// txCtxKey defines a context key type to store stransaction.
type txCtxKey string

// TxKey defines the key to access transaction stored in the context.
const TxKey txCtxKey = "DBTransaction"

// DB defines an sql database interface for the service.
type DB interface {
	// SQLDB returns DB's instance in default SQL format.
	SQLDB() *sql.DB
	// BeginTx returns a new context (being a child of the passed one)
	// with a newly started transaction inside stored by TxKey.
	BeginTx(context.Context, *sql.TxOptions) (context.Context, error)
	// CommitTx is used to commit the transaction stored in the context.
	CommitTx(context.Context) error
	// WithTx(func(context.Context) error) error
	// PingContext pings the DB.
	PingContext(context.Context) error
	// QueryOneContext is used to fetch one row from the DB and unmarshal it into dst.
	QueryOneContext(ctx context.Context, dst any, query string, args ...any) error
	// QueryManyContext is used to fetch many rows from
	// the DB and unmarshal it into dst which should be a slice type.
	QueryManyContext(ctx context.Context, dst any, query string, args ...any) error
	// ExecContext executes the passed query. MUST cancel the transaction if fails and transaction is present in the context.
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}
