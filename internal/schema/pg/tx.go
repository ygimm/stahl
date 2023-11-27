package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type Tx *sqlx.Tx

type txFunc func(ctx context.Context, tx *sqlx.Tx) error

func WithTx(ctx context.Context, db *sqlx.DB, fn txFunc, opts *sql.TxOptions) (err error) {
	tx, err := db.Beginx()
	if err != nil {
		return fmt.Errorf("cannot begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			rollbackTx(ctx, tx)
			panic(p)
		} else if err != nil {
			rollbackTx(ctx, tx)
		} else {
			err = tx.Commit()
			if err != nil {
				err = fmt.Errorf("cannot commit transaction: %w", err)
			}
		}
	}()

	err = fn(ctx, tx)

	return err
}

func rollbackTx(ctx context.Context, tx *sqlx.Tx) {
	if err := tx.Rollback(); err != nil {
		if !errors.Is(err, pq.ErrChannelAlreadyOpen) {
			log.Default().Printf("tx.Rollback(): %v", err)
		}
	}
}
