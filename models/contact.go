package models

import "time"

// Contact represents a vendor or customer.
type Contact struct {
	ID              int       `json:"id"`
	Name            string    `json:"name"`
	Type            string    `json:"type"` // vendor, customer
	Email           *string   `json:"email"`
	Phone           *string   `json:"phone"`
	TotalAmount     int       `json:"total_amount"`     // Computed: Sum of bills/invoices
	AllocatedAmount int       `json:"allocated_amount"` // Computed: Sum of payments
	Balance         int       `json:"balance"`          // Computed: Total - Allocated
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ContactInput is used for creating/updating contacts.
type ContactInput struct {
	Name  string  `json:"name"`
	Type  string  `json:"type"`
	Email *string `json:"email"`
	Phone *string `json:"phone"`
}

func (c *ContactInput) Validate() string {
	if c.Name == "" {
		return "name is required"
	}
	switch c.Type {
	case "vendor", "customer":
	default:
		return "type must be one of: vendor, customer"
	}
	return ""
}
