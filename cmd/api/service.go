package main

import (
	"context"

	payment "tx-api"
)

type Service interface {
	Payment(ctx context.Context, id string) (*payment.Payment, error)
	User(ctx context.Context, id string) (*payment.User, error)
}
