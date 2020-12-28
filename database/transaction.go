package transaction

import (
	"context"
	"database/sql"
	"fmt"
)

func Do(db *sql.DB, f func(*sql.Tx) error) error {
	ctx := context.TODO()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	err = f(tx)
	if err != nil {
		if txErr := tx.Rollback(); txErr != nil {
			return fmt.Errorf("Underlying: %v, %w", err, txErr)
		}
		return err
	}

	return tx.Commit()
}
