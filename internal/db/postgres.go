package db

import (
	"context"
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

// Postgres is a simple wrapper over the sql.DB connector.
type Postgres struct {
	*sqlx.DB
}

// NewPostgres creates a Postgres wrapper.
func NewPostgres(ctx context.Context, address string) (*Postgres, error) {
	db, err := sqlx.Open("pgx", address)
	if err != nil {
		return nil, err
	}
	return &Postgres{
		DB: db,
	}, nil
}

// SQLDB returns DB connection instance as an sql.DB pointer.
func (p *Postgres) SQLDB() *sql.DB {
	return p.DB.DB
}

// PingContext pings the DB.
func (p *Postgres) PingContext(ctx context.Context) error {
	return p.DB.PingContext(ctx)
}

// QueryOneContext executes the single-row SELECT query and unmarshals its retults into dst.
// Would use the transcation if present in the context.
func (p *Postgres) QueryOneContext(ctx context.Context, dst any, query string, args ...any) error {
	var (
		tx, ok     = ctx.Value(TxKey).(*sqlx.Tx)
		selectFunc func(context.Context, string, ...any) *sqlx.Row
	)

	if !ok || tx == nil {
		selectFunc = p.DB.QueryRowxContext
	} else {
		selectFunc = tx.QueryRowxContext
	}

	return selectFunc(ctx, query, args...).StructScan(dst)
}

// QueryManyContext executes the multi-row SELECT query and unmarshals its retults
// into dst which should be a slice type.
// Would use the transcation if present in the context.
func (p *Postgres) QueryManyContext(ctx context.Context, dst any, query string, args ...any) error {
	var (
		tx, ok     = ctx.Value(TxKey).(*sqlx.Tx)
		selectFunc func(context.Context, any, string, ...any) error
	)

	if !ok || tx == nil {
		selectFunc = p.DB.SelectContext
	} else {
		selectFunc = tx.GetContext
	}

	return selectFunc(ctx, dst, query, args...)
}

// ExecContext executes the passed query. Would use the transcation if present in the context.
func (p *Postgres) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	var (
		r        sql.Result
		err      error
		tx, ok   = ctx.Value(TxKey).(*sqlx.Tx)
		execFunc func(context.Context, string, ...any) (sql.Result, error)
	)

	if !ok || tx == nil {
		execFunc = p.DB.ExecContext
	} else {
		execFunc = tx.ExecContext
	}

	r, err = execFunc(ctx, query, args...)

	if err != nil && ok && tx != nil {
		tx.Rollback()
	}

	return r, err
}

// BeginTx creates a new transcation and puts it into the returned context.
func (p *Postgres) BeginTx(ctx context.Context, opts *sql.TxOptions) (context.Context, error) {
	var (
		t      *sqlx.Tx
		newCtx context.Context
		err    error
	)

	t, err = p.DB.BeginTxx(ctx, opts)
	if err != nil {
		return newCtx, err
	}

	return context.WithValue(ctx, TxKey, t), nil
}

// CommitTx commits the transaction inside the passed context.
func (p *Postgres) CommitTx(ctx context.Context) error {
	tx, ok := ctx.Value(TxKey).(*sqlx.Tx)
	if !ok || tx == nil {
		return ErrNoTransactionInCtx
	}
	return tx.Commit()
}
