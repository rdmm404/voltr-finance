package transaction

type TransactionType uint8

const (
	TransactionTypePersonal TransactionType = 1
	TransactionTypeHousehold  TransactionType = 2
)
