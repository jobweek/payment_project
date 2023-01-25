package payment

const (
	WithdrawTx = "withdraw"
	Deposit    = "deposit"
)

type User struct {
	ID      string
	Balance int
}

type Payment struct {
	ID     string
	UserID string
	Amount int
	TxType string
}
