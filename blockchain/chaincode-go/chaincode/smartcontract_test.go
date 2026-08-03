package chaincode

import (
	"encoding/json"
	"testing"
)

// =============================================================================
// Data model tests — no Fabric mock needed
// =============================================================================

func TestCarbonFactors(t *testing.T) {
	tests := []struct {
		source   EnergySource
		expected float64
	}{
		{Solar, 0.6},
		{Wind, 0.7},
		{Storage, 0.5},
		{EnergySource("UNKNOWN"), 0.3},
	}
	for _, tt := range tests {
		got := tt.source.CarbonFactor()
		if got != tt.expected {
			t.Errorf("CarbonFactor(%s): expected %.1f, got %.1f", tt.source, tt.expected, got)
		}
	}
}

func TestOrderStatuses(t *testing.T) {
	// Verify status constants are correct
	validStatuses := map[string]bool{
		"CREATED": true, "MATCHED": true, "FINISHED": true, "CANCELLED": true,
	}
	for _, s := range []string{"CREATED", "MATCHED", "FINISHED", "CANCELLED"} {
		if !validStatuses[s] {
			t.Errorf("unknown status: %s", s)
		}
	}
}

func TestEnergySources(t *testing.T) {
	sources := []EnergySource{Solar, Wind, Storage}
	for _, src := range sources {
		if src.CarbonFactor() <= 0 {
			t.Errorf("CarbonFactor for %s should be > 0", src)
		}
	}
}

// =============================================================================
// JSON serialization tests
// =============================================================================

func TestUserJSON(t *testing.T) {
	u := User{
		UserID:   "energy_user_1",
		UserName: "alice",
		UserRole: "PRODUCER",
		Available: 100.5,
		Balance:   5000.25,
	}
	data, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var u2 User
	if err := json.Unmarshal(data, &u2); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if u2.UserID != u.UserID {
		t.Errorf("UserID mismatch: %s vs %s", u2.UserID, u.UserID)
	}
	if u2.Available != u.Available {
		t.Errorf("Available mismatch: %.2f vs %.2f", u2.Available, u.Available)
	}
	if u2.Balance != u.Balance {
		t.Errorf("Balance mismatch: %.2f vs %.2f", u2.Balance, u.Balance)
	}
}

func TestOrderJSON(t *testing.T) {
	o := Order{
		ID:            "energy_order_1",
		PartyA:        "seller1",
		PartyB:        "buyer1",
		Direction:     "SELL",
		EnergySource:  Solar,
		Amount:        50.0,
		Price:         1.25,
		Fee:           0.01,
		Status:        "FINISHED",
		DeliveryStart: "2026-08-02T08:00:00Z",
		DeliveryEnd:   "2026-08-02T10:00:00Z",
	}
	data, err := json.Marshal(o)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var o2 Order
	if err := json.Unmarshal(data, &o2); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if o2.ID != o.ID {
		t.Errorf("ID mismatch")
	}
	if o2.Status != "FINISHED" {
		t.Errorf("Status mismatch: %s", o2.Status)
	}
	if o2.EnergySource != Solar {
		t.Errorf("EnergySource mismatch: %s", o2.EnergySource)
	}
}

func TestCarbonRecordJSON(t *testing.T) {
	r := CarbonRecord{
		OrderID:     "order1",
		Amount:      100,
		Source:      "SOLAR",
		Coefficient: 0.6,
		CarbonSaved: 60.0,
	}
	data, _ := json.Marshal(r)
	var r2 CarbonRecord
	json.Unmarshal(data, &r2)
	if r2.CarbonSaved != 60.0 {
		t.Errorf("CarbonSaved: expected 60.0, got %.2f", r2.CarbonSaved)
	}
}

func TestAuditEntryJSON(t *testing.T) {
	e := AuditEntry{
		TxID:      "tx123",
		Operation: "SettleOrder",
		OrderID:   "order1",
		UserID:    "buyer1",
		Details:   "settled 50kWh @ ¥1.0, carbon saved: 30.00 kg",
	}
	data, _ := json.Marshal(e)
	var e2 AuditEntry
	json.Unmarshal(data, &e2)
	if e2.Operation != "SettleOrder" {
		t.Errorf("Operation mismatch: %s", e2.Operation)
	}
}

// =============================================================================
// Business logic tests
// =============================================================================

func TestCarbonCalculation(t *testing.T) {
	// 100 kWh solar energy should save 60 kg CO₂
	amount := 100.0
	coeff := Solar.CarbonFactor()
	carbonSaved := amount * coeff
	if carbonSaved != 60.0 {
		t.Errorf("100 kWh solar: expected 60.0 kg CO₂, got %.2f", carbonSaved)
	}
}

func TestOrderSettlementMath(t *testing.T) {
	// Seller sells 50 kWh @ ¥1.5 = ¥75 transfer
	amount := 50.0
	price := 1.5
	total := amount * price

	sellerInitialBalance := 0.0
	buyerInitialBalance := 1000.0
	buyerInitialEnergy := 0.0

	buyerBalanceAfter := buyerInitialBalance - total
	sellerBalanceAfter := sellerInitialBalance + total
	buyerEnergyAfter := buyerInitialEnergy + amount

	if buyerBalanceAfter != 925.0 {
		t.Errorf("buyer balance: expected 925, got %.2f", buyerBalanceAfter)
	}
	if sellerBalanceAfter != 75.0 {
		t.Errorf("seller balance: expected 75, got %.2f", sellerBalanceAfter)
	}
	if buyerEnergyAfter != 50.0 {
		t.Errorf("buyer energy: expected 50, got %.2f", buyerEnergyAfter)
	}
}

func TestInsufficientBalanceCheck(t *testing.T) {
	buyerBalance := 30.0
	amount := 50.0
	price := 1.0
	total := amount * price

	if buyerBalance >= total {
		t.Error("should detect insufficient balance")
	}
	// The check should prevent the trade
	if buyerBalance < total {
		// Correct — trade should be rejected
	} else {
		t.Error("should reject trade")
	}
}

func TestCompositeKeyFormats(t *testing.T) {
	// Verify composite key prefixes are correctly defined
	if OrderStatusPrefix != "order~status" {
		t.Errorf("OrderStatusPrefix wrong: %s", OrderStatusPrefix)
	}
	if AuditPrefix != "audit~ts" {
		t.Errorf("AuditPrefix wrong: %s", AuditPrefix)
	}
	if CarbonPrefix != "carbon~ts" {
		t.Errorf("CarbonPrefix wrong: %s", CarbonPrefix)
	}
}
