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
	r.Put("/api/v1/bills/{id}", UpdateBill)
	r.Delete("/api/v1/bills/{id}", DeleteBill)
	r.Get("/api/v1/bills/{id}/items", ListBillItems)
	r.Post("/api/v1/bills/{id}/items", CreateBillItem)
	r.Put("/api/v1/bills/{id}/items/{itemId}", UpdateBillItem)
	r.Delete("/api/v1/bills/{id}/items/{itemId}", DeleteBillItem)

	// Register invoice item routes
	r.Post("/api/v1/invoices", CreateInvoice)
	r.Get("/api/v1/invoices/{id}", GetInvoice)
	r.Put("/api/v1/invoices/{id}", UpdateInvoice)
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

	// Create an item with unit and client-supplied amount.
	status, resp = apiRequest(t, r, "POST", fmt.Sprintf("/api/v1/bills/%d/items", billID), map[string]interface{}{
		"description": "Widget A",
		"quantity":    2.0,
		"unit":        "pcs",
		"unit_price":  50.0,  // 50 rupees = 5000 paise
		"amount":      100.0, // client-supplied: 100 rupees = 10000 paise
	})
	if status != http.StatusCreated {
		t.Fatalf("create bill item: status %d, error %v", status, resp["error"])
	}
	itemData := resp["data"].(map[string]interface{})
	itemID := int(itemData["id"].(float64))
	if itemData["description"] != "Widget A" {
		t.Errorf("expected description 'Widget A', got %v", itemData["description"])
	}
	if itemData["unit"] != "pcs" {
		t.Errorf("expected unit 'pcs', got %v", itemData["unit"])
	}
	// amount should be the client-supplied value: 100 rupees = 10000 paise
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

	// Update the item with a different client-supplied amount and no unit.
	status, resp = apiRequest(t, r, "PUT", fmt.Sprintf("/api/v1/bills/%d/items/%d", billID, itemID), map[string]interface{}{
		"description": "Widget A Updated",
		"quantity":    3.0,
		"unit_price":  50.0,
		"amount":      150.0, // client-supplied: 150 rupees = 15000 paise
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
		"amount":      100.0,
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
			input:  map[string]interface{}{"quantity": 1.0, "unit_price": 10.0, "amount": 10.0},
			wantOK: false,
		},
		{
			name:   "zero quantity",
			input:  map[string]interface{}{"description": "item", "quantity": 0.0, "unit_price": 10.0, "amount": 10.0},
			wantOK: false,
		},
		{
			name:   "negative quantity",
			input:  map[string]interface{}{"description": "item", "quantity": -1.0, "unit_price": 10.0, "amount": 10.0},
			wantOK: false,
		},
		{
			name:   "missing amount",
			input:  map[string]interface{}{"description": "item", "quantity": 1.0, "unit_price": 10.0},
			wantOK: false,
		},
		{
			name:   "valid item without unit",
			input:  map[string]interface{}{"description": "item", "quantity": 1.0, "unit_price": 10.0, "amount": 10.0},
			wantOK: true,
		},
		{
			name:   "valid item with unit",
			input:  map[string]interface{}{"description": "item", "quantity": 1.0, "unit": "kg", "unit_price": 10.0, "amount": 10.0},
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

// TestBillCreateAndUpdateWithItems verifies that bills can be created and updated with inline items.
func TestBillCreateAndUpdateWithItems(t *testing.T) {
	r, cleanup := setupItemsTestRouter(t)
	defer cleanup()

	createStatus, createResp := apiRequest(t, r, "POST", "/api/v1/bills", map[string]interface{}{
		"bill_number": "BILL-INLINE",
		"amount":      200.0,
		"status":      "draft",
		"items": []map[string]interface{}{
			{"description": "Item A", "quantity": 2.0, "unit_price": 50.0, "amount": 100.0},
			{"description": "Item B", "quantity": 1.0, "unit_price": 100.0, "amount": 100.0},
		},
	})
	if createStatus != http.StatusCreated {
		t.Fatalf("create bill with items: status %d, error %v", createStatus, createResp["error"])
	}
	billData := createResp["data"].(map[string]interface{})
	billID := int(billData["id"].(float64))
	items := billData["items"].([]interface{})
	if len(items) != 2 {
		t.Fatalf("expected 2 items on create response, got %d", len(items))
	}
	first := items[0].(map[string]interface{})
	if int(first["amount"].(float64)) != 10000 {
		t.Fatalf("expected first item amount 10000, got %v", first["amount"])
	}

	status, listResp := apiRequest(t, r, "GET", fmt.Sprintf("/api/v1/bills/%d/items", billID), nil)
	if status != http.StatusOK {
		t.Fatalf("list items after create: status %d, error %v", status, listResp["error"])
	}
	if len(listResp["data"].([]interface{})) != 2 {
		t.Fatalf("expected 2 stored items, got %d", len(listResp["data"].([]interface{})))
	}

	updateStatus, updateResp := apiRequest(t, r, "PUT", fmt.Sprintf("/api/v1/bills/%d", billID), map[string]interface{}{
		"bill_number": "BILL-INLINE",
		"amount":      300.0,
		"status":      "received",
		"items": []map[string]interface{}{
			{"description": "Updated Item", "quantity": 3.0, "unit_price": 100.0, "amount": 300.0},
		},
	})
	if updateStatus != http.StatusOK {
		t.Fatalf("update bill with items: status %d, error %v", updateStatus, updateResp["error"])
	}
	updatedBill := updateResp["data"].(map[string]interface{})
	updatedItems := updatedBill["items"].([]interface{})
	if len(updatedItems) != 1 {
		t.Fatalf("expected 1 item after update, got %d", len(updatedItems))
	}
	updated := updatedItems[0].(map[string]interface{})
	if updated["description"] != "Updated Item" {
		t.Fatalf("expected updated description, got %v", updated["description"])
	}
	if int(updated["amount"].(float64)) != 30000 {
		t.Fatalf("expected updated amount 30000, got %v", updated["amount"])
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

	// Create an item with unit and client-supplied amount.
	status, resp = apiRequest(t, r, "POST", fmt.Sprintf("/api/v1/invoices/%d/items", invoiceID), map[string]interface{}{
		"description": "Consulting",
		"quantity":    4.0,
		"unit":        "hrs",
		"unit_price":  25.0,  // 25 rupees = 2500 paise
		"amount":      100.0, // client-supplied: 100 rupees = 10000 paise
	})
	if status != http.StatusCreated {
		t.Fatalf("create invoice item: status %d, error %v", status, resp["error"])
	}
	itemData := resp["data"].(map[string]interface{})
	itemID := int(itemData["id"].(float64))
	if itemData["unit"] != "hrs" {
		t.Errorf("expected unit 'hrs', got %v", itemData["unit"])
	}
	// amount should be client-supplied value: 100 rupees = 10000 paise
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

	// Update the item with a different client-supplied amount.
	status, resp = apiRequest(t, r, "PUT", fmt.Sprintf("/api/v1/invoices/%d/items/%d", invoiceID, itemID), map[string]interface{}{
		"description": "Consulting Updated",
		"quantity":    5.0,
		"unit":        "hrs",
		"unit_price":  25.0,
		"amount":      125.0, // client-supplied: 125 rupees = 12500 paise
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
		"amount":      200.0,
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

// TestInvoiceCreateAndUpdateWithItems verifies that invoices can be created and updated with inline items.
func TestInvoiceCreateAndUpdateWithItems(t *testing.T) {
	r, cleanup := setupItemsTestRouter(t)
	defer cleanup()

	createStatus, createResp := apiRequest(t, r, "POST", "/api/v1/invoices", map[string]interface{}{
		"invoice_number": "INV-INLINE",
		"amount":         150.0,
		"status":         "draft",
		"items": []map[string]interface{}{
			{"description": "Service", "quantity": 3.0, "unit_price": 50.0, "amount": 150.0},
		},
	})
	if createStatus != http.StatusCreated {
		t.Fatalf("create invoice with items: status %d, error %v", createStatus, createResp["error"])
	}
	invoiceData := createResp["data"].(map[string]interface{})
	invoiceID := int(invoiceData["id"].(float64))
	items := invoiceData["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("expected 1 item on invoice create, got %d", len(items))
	}
	if int(items[0].(map[string]interface{})["amount"].(float64)) != 15000 {
		t.Fatalf("expected item amount 15000, got %v", items[0].(map[string]interface{})["amount"])
	}

	updateStatus, updateResp := apiRequest(t, r, "PUT", fmt.Sprintf("/api/v1/invoices/%d", invoiceID), map[string]interface{}{
		"invoice_number": "INV-INLINE",
		"amount":         200.0,
		"status":         "sent",
		"items": []map[string]interface{}{
			{"description": "Updated Service", "quantity": 4.0, "unit_price": 50.0, "amount": 200.0},
			{"description": "Add-on", "quantity": 1.0, "unit_price": 50.0, "amount": 50.0},
		},
	})
	if updateStatus != http.StatusOK {
		t.Fatalf("update invoice with items: status %d, error %v", updateStatus, updateResp["error"])
	}
	updatedInvoice := updateResp["data"].(map[string]interface{})
	updatedItems := updatedInvoice["items"].([]interface{})
	if len(updatedItems) != 2 {
		t.Fatalf("expected 2 items after invoice update, got %d", len(updatedItems))
	}
	if updatedItems[0].(map[string]interface{})["description"] != "Updated Service" {
		t.Fatalf("unexpected updated item description: %v", updatedItems[0].(map[string]interface{})["description"])
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
		"description": "item", "quantity": 1.0, "unit_price": 10.0, "amount": 10.0,
	})
	if status != http.StatusNotFound {
		t.Errorf("expected 404 creating item on non-existent bill, got %d", status)
	}
}
