package postgres

import (
	"context"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	ppostgres "github.com/dz-market/platform/database/postgres"
)

var _ ppostgres.Querier = (*TxQuerier)(nil)

type TxQuerier struct {
	pool   *pgxpool.Pool
	getter *trmpgx.CtxGetter
}

func NewTxQuerier(pool *pgxpool.Pool, getter *trmpgx.CtxGetter) *TxQuerier {
	return &TxQuerier{
		pool:   pool,
		getter: getter,
	}
}

func (q *TxQuerier) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	return q.getter.DefaultTrOrDB(ctx, q.pool).Exec(ctx, query, args...)
}

func (q *TxQuerier) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	return q.getter.DefaultTrOrDB(ctx, q.pool).Query(ctx, query, args...)
}

func (q *TxQuerier) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return q.getter.DefaultTrOrDB(ctx, q.pool).QueryRow(ctx, query, args...)
}