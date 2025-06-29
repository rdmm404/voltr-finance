package transaction

import "fmt"

func SaveTransactions(transactions []Transaction) error {
	_, err := fmt.Printf("received transactions %v\n", transactions)
	if err != nil {
		return err
	}
	return nil
}
