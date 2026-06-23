package common

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func ExecuteTransaction(ctx context.Context, db *sqlx.DB, fn func(tx *sqlx.Tx) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	defer func() {
		if pErr := recover(); pErr != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				fmt.Printf("failed to rollback transaction after panic: %v", rollbackErr)
			}
			panic(pErr)
		}

		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				err = fmt.Errorf("transaction failed: %w, rollback failed: %v", err, rollbackErr)
			}
		} else {
			if commitErr := tx.Commit(); commitErr != nil {
				err = fmt.Errorf("failed to commit transaction: %w", commitErr)
			}
		}
	}()

	err = fn(tx)
	return err
}

func GetEmployeeId(c *gin.Context) (uuid.UUID, error) {
	empIDRaw, ok := c.Get("user_id")
	if !ok {
		return uuid.Nil, errors.New("employee ID missing")
	}

	empIDStr, ok := empIDRaw.(string)
	if !ok {
		return uuid.Nil, errors.New("invalid employee ID format")
	}

	empID, err := uuid.Parse(empIDStr)
	if err != nil {
		return uuid.Nil, errors.New("invalid employee UUID")
	}

	return empID, nil
}

func GetMonthYear(c *gin.Context) (int, int, error) {
	month, err := strconv.Atoi(
		c.DefaultQuery("month", fmt.Sprintf("%d", int(time.Now().Month()))),
	)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid month")
	}

	year, err := strconv.Atoi(
		c.DefaultQuery("year", fmt.Sprintf("%d", time.Now().Year())),
	)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid year")
	}

	return month, year, nil
}
