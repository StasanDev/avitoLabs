package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txContextKey struct{}

type TxManager struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

func NewTxManager(pool *pgxpool.Pool, queryTimeout time.Duration) *TxManager {
	return &TxManager{
		pool:         pool,
		queryTimeout: queryTimeout,
	}
}

func (m *TxManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := txFromContext(ctx); ok {
		return fn(ctx)
	}

	beginCtx, cancelBegin := context.WithTimeout(ctx, m.queryTimeout)
	tx, err := m.pool.BeginTx(beginCtx, pgx.TxOptions{
		IsoLevel: pgx.ReadCommitted,
	})
	cancelBegin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	rollbackBaseCtx := context.WithoutCancel(ctx)
	committed := false
	defer func() {
		if !committed {
			rollbackCtx, cancelRollback := context.WithTimeout(rollbackBaseCtx, m.queryTimeout)
			defer cancelRollback()
			_ = tx.Rollback(rollbackCtx)
		}
	}()

	ctx = context.WithValue(ctx, txContextKey{}, tx)

	if err := fn(ctx); err != nil {
		return err
	}
	commitCtx, cancelCommit := context.WithTimeout(ctx, m.queryTimeout)
	err = tx.Commit(commitCtx)
	cancelCommit()
	if err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	committed = true
	return nil
}

func txFromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txContextKey{}).(pgx.Tx)
	return tx, ok
}
