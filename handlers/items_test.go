package handlers

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
)

func setupItemsTestRouter(t *testing.T) (*chi.Mux, func()) {
	t.Helper()
	r, cleanup := setupTestRouter(t)

	// Register bill item routes
	r.Post("/api/v1/bills", CreateBill)
	r.Get("/api/v1/bills/{id}", GetBill)
	r.Delete("/api/v1/bills/{id}", DeleteBill)
	r.Get("/api/v1/bills/{id}/items", ListBillItems)
	r.Post("/api/v1/bills/{id}/items", CreateBillItem)
	r.Put("/api/v1/bills/{id}/items/{itemId}", UpdateBillItem)
	r.Delete("/api/v1/bills/{id}/items/{itemId}", DeleteBillItem)

	// Register invoice item routes
	r.Post("/api/v1/invoices", CreateInvoice)
	r.Get("/api/v1/invoices/{id}", GetInvoice)
	r.Delete("/api/v1/invoices/{id}", DeleteInvoice)
	r.Get("/api/v1/invoices/{id}/items", ListInvoiceItems)
	r.Post("/api/v1/invoices/{id}/items", CreateInvoiceItem)
	r.Put("/api/v1/invoices/{id}/items/{itemId}", UpdateInvoiceItem)
	r.Delete("/api/v1/invoices/{id}/items/{itemId}", DeleteInvoiceItem)

	return r, cleanup
}

func createTestBill(t *testing.T, r http.Handler) int {
	t.Helper()
	status, resp := apiRequest(t, r, "POST", "/api/v1/bills", map[string]interface{}{
		"bill_number": "BILL-001",
		"amount":      100.0,
		"status":      "draft",
	})
	if status != http.StatusCreated {
		t.Fatalf("create bill: status %d, error %v", status, resp["error"])
	}
	return int(resp["data"].(map[string]interface{})["id"].(float64))
}

func createTestInvoice(t *testing.T, r http.Handler) int {
	t.Helper()
	status, resp := apiRequest(t, r, "POST", "/api/v1/invoices", map[string]interface{}{
		"invoice_number": "INV-001",
		"amount":         200.0,
		"status":         "draft",
	})
	if status != http.StatusCreated {
		t.Fatalf("create invoice: status %d, error %v", status, resp["error"])
	}
	return int(resp["data"].(map[string]interface{})["id"].(float64))
}

// TestBillItemsCRUD tests create, list, update, and delete of bill line items.
func TestBillItemsCRUD(t *testing.T) {
	r, cleanup := setupItemsTestRouter(t)
	defer cleanup()

	billID := createTestBill(t, r)

	// Initially no items.
	status, resp := apiRequest(t, r, "GET", fmt.Sprintf("/api/v1/bills/%d/items", billID), nil)
	if status != http.StatusOK {
		t.Fatalf("list bill items: status %d", status)
	}
	items := resp["data"].([]interface{})
	if len(items) != 0 {
		t.Errorf("expected 0 items initially, got %d", len(items))
	}

	// Create an item.
	status, resp = apiRequest(t, r, "POST", fmt.Sprintf("/api/v1/bills/%d/items", billID), map[string]interface{}{
		"description": "Widget A",
		"quantity":    2.0,
		"unit_price":  50.0, // 50 rupees = 5000 paise
	})
	if status != http.StatusCreated {
		t.Fatalf("create bill item: status %d, error %v", status, resp["error"])
	}
	itemData := resp["data"].(map[string]interface{})
	itemID := int(itemData["id"].(float64))
	if itemData["description"] != "Widget A" {
		t.Errorf("expected description 'Widget A', got %v", itemData["description"])
	}
	// amount should be quantity * unit_price = 2 * 5000 = 10000 paise
	if int(itemData["amount"].(float64)) != 10000 {
		t.Errorf("expected amount 10000, got %v", itemData["amount"])
	}

	// List items now shows the created item.
	status, resp = apiRequest(t, r, "GET", fmt.Sprintf("/api/v1/bills/%d/items", billID), nil)
	if status != http.StatusOK {
		t.Fatalf("list bill items: status %d", status)
	}
	items = resp["data"].([]interface{})
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}

	// GET /bills/{id} includes items.
	status, resp = apiRequest(t, r, "GET", fmt.Sprintf("/api/v1/bills/%d", billID), nil)
	if status != http.StatusOK {
		t.Fatalf("get bill: status %d", status)
	}
	billData := resp["data"].(map[string]interface{})
	billItems := billData["items"].([]interface{})
	if len(billItems) != 1 {
		t.Errorf("expected 1 item in bill response, got %d", len(billItems))
	}

	// Update the item.
	status, resp = apiRequest(t, r, "PUT", fmt.Sprintf("/api/v1/bills/%d/items/%d", billID, itemID), map[string]interface{}{
		"description": "Widget A Updated",
		"quantity":    3.0,
		"unit_price":  50.0,
	})
	if status != http.StatusOK {
		t.Fatalf("update bill item: status %d, error %v", status, resp["error"])
	}
	updatedItem := resp["data"].(map[string]interface{})
	if updatedItem["description"] != "Widget A Updated" {
		t.Errorf("expected updated description, got %v", updatedItem["description"])
	}
	if int(updatedItem["amount"].(float64)) != 15000 {
		t.Errorf("expected updated amount 15000, got %v", updatedItem["amount"])
	}

	// Delete the item.
	status, resp = apiRequest(t, r, "DELETE", fmt.Sprintf("/api/v1/bills/%d/items/%d", billID, itemID), nil)
	if status != http.StatusOK {
		t.Fatalf("delete bill item: status %d, error %v", status, resp["error"])
	}

	// List items now empty.
	status, resp = apiRequest(t, r, "GET", fmt.Sprintf("/api/v1/bills/%d/items", billID), nil)
	if status != http.StatusOK {
		t.Fatalf("list bill items after delete: status %d", status)
	}
	items = resp["data"].([]interface{})
	if len(items) != 0 {
		t.Errorf("expected 0 items after delete, got %d", len(items))
	}
}

// TestBillItemsDeletedWithBill verifies that items are removed when a bill is deleted.
func TestBillItemsDeletedWithBill(t *testing.T) {
	r, cleanup := setupItemsTestRouter(t)
	defer cleanup()

	billID := createTestBill(t, r)

	// Add an item.
	apiRequest(t, r, "POST", fmt.Sprintf("/api/v1/bills/%d/items", billID), map[string]interface{}{
		"description": "Service fee",
		"quantity":    1.0,
		"unit_price":  100.0,
	})

	// Delete the bill.
	status, resp := apiRequest(t, r, "DELETE", fmt.Sprintf("/api/v1/bills/%d", billID), nil)
	if status != http.StatusOK {
		t.Fatalf("delete bill: status %d, error %v", status, resp["error"])
	}

	// GET /bills/{id}/items on deleted bill returns 404.
	status, _ = apiRequest(t, r, "GET", fmt.Sprintf("/api/v1/bills/%d/items", billID), nil)
	if status != http.StatusNotFound {
		t.Errorf("expected 404 for items of deleted bill, got %d", status)
	}
}

// TestBillItemValidation tests input validation for bill items.
func TestBillItemValidation(t *testing.T) {
	r, cleanup := setupItemsTestRouter(t)
	defer cleanup()

	billID := createTestBill(t, r)

	tests := []struct {
		name   string
		input  map[string]interface{}
		wantOK bool
	}{
		{
			name:   "missing description",
			input:  map[string]interface{}{"quantity": 1.0, "unit_price": 10.0},
			wantOK: false,
		},
		{
			name:   "zero quantity",
			input:  map[string]interface{}{"description": "item", "quantity": 0.0, "unit_price": 10.0},
			wantOK: false,
		},
		{
			name:   "negative quantity",
			input:  map[string]interface{}{"description": "item", "quantity": -1.0, "unit_price": 10.0},
			wantOK: false,
		},
		{
			name:   "valid item",
			input:  map[string]interface{}{"description": "item", "quantity": 1.0, "unit_price": 10.0},
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, _ := apiRequest(t, r, "POST", fmt.Sprintf("/api/v1/bills/%d/items", billID), tt.input)
			if tt.wantOK && status != http.StatusCreated {
				t.Errorf("expected 201, got %d", status)
			}
			if !tt.wantOK && status != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", status)
			}
		})
	}
}

// TestInvoiceItemsCRUD tests create, list, update, and delete of invoice line items.
func TestInvoiceItemsCRUD(t *testing.T) {
	r, cleanup := setupItemsTestRouter(t)
	defer cleanup()

	invoiceID := createTestInvoice(t, r)

	// Initially no items.
	status, resp := apiRequest(t, r, "GET", fmt.Sprintf("/api/v1/invoices/%d/items", invoiceID), nil)
	if status != http.StatusOK {
		t.Fatalf("list invoice items: status %d", status)
	}
	items := resp["data"].([]interface{})
	if len(items) != 0 {
		t.Errorf("expected 0 items initially, got %d", len(items))
	}

	// Create an item.
	status, resp = apiRequest(t, r, "POST", fmt.Sprintf("/api/v1/invoices/%d/items", invoiceID), map[string]interface{}{
		"description": "Consulting",
		"quantity":    4.0,
		"unit_price":  25.0, // 25 rupees = 2500 paise
	})
	if status != http.StatusCreated {
		t.Fatalf("create invoice item: status %d, error %v", status, resp["error"])
	}
	itemData := resp["data"].(map[string]interface{})
	itemID := int(itemData["id"].(float64))
	// amount = 4 * 2500 = 10000 paise
	if int(itemData["amount"].(float64)) != 10000 {
		t.Errorf("expected amount 10000, got %v", itemData["amount"])
	}

	// GET /invoices/{id} includes items.
	status, resp = apiRequest(t, r, "GET", fmt.Sprintf("/api/v1/invoices/%d", invoiceID), nil)
	if status != http.StatusOK {
		t.Fatalf("get invoice: status %d", status)
	}
	invData := resp["data"].(map[string]interface{})
	invItems := invData["items"].([]interface{})
	if len(invItems) != 1 {
		t.Errorf("expected 1 item in invoice response, got %d", len(invItems))
	}

	// Update the item.
	status, resp = apiRequest(t, r, "PUT", fmt.Sprintf("/api/v1/invoices/%d/items/%d", invoiceID, itemID), map[string]interface{}{
		"description": "Consulting Updated",
		"quantity":    5.0,
		"unit_price":  25.0,
	})
	if status != http.StatusOK {
		t.Fatalf("update invoice item: status %d, error %v", status, resp["error"])
	}
	updatedItem := resp["data"].(map[string]interface{})
	if updatedItem["description"] != "Consulting Updated" {
		t.Errorf("expected updated description, got %v", updatedItem["description"])
	}
	if int(updatedItem["amount"].(float64)) != 12500 {
		t.Errorf("expected updated amount 12500, got %v", updatedItem["amount"])
	}

	// Delete the item.
	status, _ = apiRequest(t, r, "DELETE", fmt.Sprintf("/api/v1/invoices/%d/items/%d", invoiceID, itemID), nil)
	if status != http.StatusOK {
		t.Fatalf("delete invoice item: status %d", status)
	}

	// List items now empty.
	status, resp = apiRequest(t, r, "GET", fmt.Sprintf("/api/v1/invoices/%d/items", invoiceID), nil)
	if status != http.StatusOK {
		t.Fatalf("list invoice items after delete: status %d", status)
	}
	if len(resp["data"].([]interface{})) != 0 {
		t.Errorf("expected 0 items after delete")
	}
}

// TestInvoiceItemsDeletedWithInvoice verifies that items are removed when an invoice is deleted.
func TestInvoiceItemsDeletedWithInvoice(t *testing.T) {
	r, cleanup := setupItemsTestRouter(t)
	defer cleanup()

	invoiceID := createTestInvoice(t, r)

	apiRequest(t, r, "POST", fmt.Sprintf("/api/v1/invoices/%d/items", invoiceID), map[string]interface{}{
		"description": "Subscription",
		"quantity":    1.0,
		"unit_price":  200.0,
	})

	// Delete the invoice.
	status, resp := apiRequest(t, r, "DELETE", fmt.Sprintf("/api/v1/invoices/%d", invoiceID), nil)
	if status != http.StatusOK {
		t.Fatalf("delete invoice: status %d, error %v", status, resp["error"])
	}

	// GET /invoices/{id}/items on deleted invoice returns 404.
	status, _ = apiRequest(t, r, "GET", fmt.Sprintf("/api/v1/invoices/%d/items", invoiceID), nil)
	if status != http.StatusNotFound {
		t.Errorf("expected 404 for items of deleted invoice, got %d", status)
	}
}

// TestBillItemNotFoundWhenBillMissing verifies 404 for items on non-existent bill.
func TestBillItemNotFoundWhenBillMissing(t *testing.T) {
	r, cleanup := setupItemsTestRouter(t)
	defer cleanup()

	status, _ := apiRequest(t, r, "GET", "/api/v1/bills/99999/items", nil)
	if status != http.StatusNotFound {
		t.Errorf("expected 404 for non-existent bill, got %d", status)
	}

	status, _ = apiRequest(t, r, "POST", "/api/v1/bills/99999/items", map[string]interface{}{
		"description": "item", "quantity": 1.0, "unit_price": 10.0,
	})
	if status != http.StatusNotFound {
		t.Errorf("expected 404 creating item on non-existent bill, got %d", status)
	}
}
