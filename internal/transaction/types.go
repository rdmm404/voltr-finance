package transaction

type TransactionType int

const (
	TransactionTypePersonal TransactionType = 1
	TransactionTypeHousehold  TransactionType = 2
)
