package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	payment "tx-api"
)

var (
	errExceedUserBalance  = fmt.Errorf("exceeded user balance")
	errUnknownPaymentType = fmt.Errorf("unknown payment type")
)

type Payment struct {
	ID     string `db:"id"`
	UserID string `db:"user_id"`
	Amount int    `db:"amount"`
	TxType string `db:"tx_type"`
}

type User struct {
	ID      string `db:"id"`
	Balance int    `db:"balance"`
}

type Service struct {
	db *sqlx.DB
}

func NewService(db *sqlx.DB) *Service {
	return &Service{
		db: db,
	}
}

func (s *Service) Payment(ctx context.Context, id string) (*payment.Payment, error) {
	var p Payment
	if err := s.db.GetContext(ctx, &p, `SELECT * FROM transactions WHERE id=$1`, id); err != nil {
		return nil, err
	}

	return &payment.Payment{
		ID:     p.ID,
		UserID: p.UserID,
		Amount: p.Amount,
		TxType: p.TxType,
	}, nil
}

func (s *Service) MakePayment(ctx context.Context, newPayment *payment.Payment) error {

	u, err := s.User(ctx, newPayment.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			u = &payment.User{
				ID: newPayment.UserID,
			}
			err := s.CreateUser(ctx, u)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	p := Payment{
		ID:     newPayment.ID,
		UserID: newPayment.UserID,
		Amount: int(newPayment.Amount),
		TxType: newPayment.TxType,
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	var newBalance int
	if newPayment.TxType == payment.WithdrawTx {
		newBalance = u.Balance - newPayment.Amount
	} else if newPayment.TxType == payment.Deposit {
		newBalance = u.Balance + newPayment.Amount
	} else {
		return errUnknownPaymentType
	}

	if newBalance < 0 {
		return errExceedUserBalance
	}

	//p.Status = payment.SuccessfulStatus

	_, err = tx.NamedExecContext(ctx, `INSERT INTO transactions (id, user_id, amount, tx_type) VALUES (:id, :user_id, :amount, :tx_type)`, &p)
	if err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			return fmt.Errorf("failed to insert payment with error %s, rollback failed with error %s", err, rollbackErr)
		}
		return fmt.Errorf("faield to insert payment with error %s", err)
	}

	_, err = tx.ExecContext(ctx, `UPDATE users SET balance=$1 WHERE id=$2`, newBalance, u.ID)
	if err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			return fmt.Errorf("failed to update user %s balance with error %s, rollback failed with error %s", u.ID, err, rollbackErr)
		}
		return fmt.Errorf("failed to update user %s balance with error %s", u.ID, err)
	}

	return tx.Commit()
}

func (s *Service) User(ctx context.Context, id string) (*payment.User, error) {
	var u User
	if err := s.db.GetContext(ctx, &u, `SELECT * FROM users WHERE id=$1`, id); err != nil {
		return nil, err
	}

	return &payment.User{
		ID:      u.ID,
		Balance: u.Balance,
	}, nil
}

func (s *Service) CreateUser(ctx context.Context, u *payment.User) error {
	_, err := s.db.NamedExecContext(ctx, `INSERT INTO users (id, balance) VALUES (:id, :balance)`, u)
	return err
}
