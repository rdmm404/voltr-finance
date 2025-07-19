package transaction

type TransactionType string

const (
	TransactionTypeCredit TransactionType = "credit"
	TransactionTypeDebit  TransactionType = "debit"
)

type Transaction struct {
	Name            string
	Description     string
	Amount          float32
	TransactionType TransactionType

}
