package models

import "time"

// Account represents a bank account, cash, or credit card.
type Account struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	Type           string    `json:"type"` // bank, cash, credit_card
	OpeningBalance int       `json:"opening_balance"`
	Balance        int       `json:"balance"` // Computed
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// AccountInput is used for creating/updating accounts.
type AccountInput struct {
	Name           string `json:"name"`
	Type           string `json:"type"`
	OpeningBalance int    `json:"opening_balance"`
}

func (a *AccountInput) Validate() string {
	if a.Name == "" {
		return "name is required"
	}
	switch a.Type {
	case "bank", "cash", "credit_card":
	default:
		return "type must be one of: bank, cash, credit_card"
	}
	return ""
}
