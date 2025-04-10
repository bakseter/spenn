package models

import (
	"gorm.io/gorm"
)

type Transaction struct {
	gorm.Model
	Amount      int
	Description string
	UserID      uint
}

func SumTransactions(transactions []Transaction) int {
	sum := 0

	for _, transaction := range transactions {
		sum += transaction.Amount
	}

	return sum
}
