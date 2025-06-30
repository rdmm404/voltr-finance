package transaction

import "fmt"

func SaveTransactions(transactions []*Transaction) error {
	fmt.Println("received transactions")
	for _, trans := range transactions {
		fmt.Printf("%+v\n", *trans)
	}
	return nil
}
