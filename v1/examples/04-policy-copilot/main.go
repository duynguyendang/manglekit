package main

import (
	"fmt"
	"log"
	"time"

	"github.com/duynguyendang/manglekit/v1/core/reflection"
)

// User represents a user in the system.
type User struct {
	Name  string
	Roles []string `mangle:"roles"`
}

// Transaction represents a financial transaction.
type Transaction struct {
	ID        string
	Amount    int64     `mangle:"amt"`
	Timestamp time.Time `mangle:"ts"`   // Tests the Time Hook
	User      *User     `mangle:"user"` // Tests Nested Struct + Pointer
}

func main() {
	// Create a sample transaction.
	tx := Transaction{
		ID:        "tx_12345",
		Amount:    1500,
		Timestamp: time.Date(2024, time.March, 15, 10, 30, 0, 0, time.UTC),
		User: &User{
			Name:  "Alice",
			Roles: []string{"admin", "editor"},
		},
	}

	// Convert the transaction to Mangle facts.
	facts, err := reflection.ToFacts("tx_1", tx)
	if err != nil {
		log.Fatalf("Failed to convert to facts: %v", err)
	}

	// Print the generated facts.
	fmt.Println("Generated Facts:")
	for _, fact := range facts {
		fmt.Println(fact.String())
	}
}
