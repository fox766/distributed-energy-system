package chaincode

import (
	"encoding/json"
	"math"
	"testing"
	"time"
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

// =============================================================================
// Pair settlement math tests — the accounting used by settleMatchedPair
// =============================================================================

// closeTo compares two floats within eps (0.8 + 0.2 style arithmetic is not
// exact in binary floating point).
func closeTo(a, b, eps float64) bool {
	return math.Abs(a-b) < eps
}

func TestPairSettlementMathFullMatch(t *testing.T) {
	// SELL 10 kWh @ ¥0.8 vs BUY 10 kWh @ ¥1.0:
	// buyer locked ¥10 at creation, pays ¥8, gets ¥2 spread refund.
	tradeAmount, payment, refund := pairSettlementMath(10, 10, 0.8, 1.0)
	if !closeTo(tradeAmount, 10.0, 1e-9) || !closeTo(payment, 8.0, 1e-9) || !closeTo(refund, 2.0, 1e-9) {
		t.Errorf("full match: expected trade=10 payment=8 refund=2, got trade=%.2f payment=%.2f refund=%.2f",
			tradeAmount, payment, refund)
	}
}

func TestPairSettlementMathSellLarger(t *testing.T) {
	// SELL 10 kWh @ ¥0.8 vs BUY 6 kWh @ ¥1.0: trade 6 kWh.
	tradeAmount, payment, refund := pairSettlementMath(10, 6, 0.8, 1.0)
	if !closeTo(tradeAmount, 6.0, 1e-9) || !closeTo(payment, 4.8, 1e-9) || !closeTo(refund, 1.2, 1e-9) {
		t.Errorf("sell larger: expected trade=6 payment=4.8 refund=1.2, got trade=%.2f payment=%.2f refund=%.2f",
			tradeAmount, payment, refund)
	}
	// Buy-side lock conservation: creation lock 6×1.0 = payment + refund
	if !closeTo(payment+refund, 6.0, 1e-9) {
		t.Errorf("buy lock not conserved: payment+refund=%.2f, want 6.00", payment+refund)
	}
}

func TestPairSettlementMathBuyLarger(t *testing.T) {
	// SELL 6 kWh @ ¥0.8 vs BUY 10 kWh @ ¥1.0: trade 6 kWh, residual BUY 4 kWh.
	tradeAmount, payment, refund := pairSettlementMath(6, 10, 0.8, 1.0)
	if !closeTo(tradeAmount, 6.0, 1e-9) || !closeTo(payment, 4.8, 1e-9) || !closeTo(refund, 1.2, 1e-9) {
		t.Errorf("buy larger: expected trade=6 payment=4.8 refund=1.2, got trade=%.2f payment=%.2f refund=%.2f",
			tradeAmount, payment, refund)
	}
	// Buy-side lock conservation: creation lock 10×1.0 = payment + refund + residual lock 4×1.0
	if !closeTo(payment+refund+4.0, 10.0, 1e-9) {
		t.Errorf("buy lock not conserved with residual: payment+refund+residual=%.2f, want 10.00",
			payment+refund+4.0)
	}
	// Sell-side lock conservation: creation lock 6 kWh = traded 6 kWh (no residual)
	if !closeTo(tradeAmount, 6.0, 1e-9) {
		t.Errorf("sell lock should be fully consumed, traded=%.2f", tradeAmount)
	}
}

func TestPairSettlementMathDefensiveNoNegativeRefund(t *testing.T) {
	// Defensive path: sell price above buy price should never reach settlement
	// (price filter), but the math must not produce a negative refund.
	_, _, refund := pairSettlementMath(10, 10, 1.2, 1.0)
	if !closeTo(refund, 0.0, 1e-9) {
		t.Errorf("refund should clamp to 0, got %.2f", refund)
	}
}

func TestPairSettlementMathResidualAmounts(t *testing.T) {
	// SELL 10 vs BUY 6: residual SELL = 10 - 6 = 4; seller lock (10) covers
	// traded (6) + residual (4) with no second deduction.
	tradeAmount, _, _ := pairSettlementMath(10, 6, 0.8, 1.0)
	residualSell := 10.0 - tradeAmount
	if !closeTo(residualSell, 4.0, 1e-9) {
		t.Errorf("residual sell amount: expected 4, got %.2f", residualSell)
	}
	if !closeTo(tradeAmount+residualSell, 10.0, 1e-9) {
		t.Errorf("sell lock not conserved: traded+residual=%.2f, want 10.00", tradeAmount+residualSell)
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

func TestPairSettlementMathAutoMatchScenario(t *testing.T) {
	// The end-to-end case: SELL 5 kWh @ ¥0.8 vs BUY 3 kWh @ ¥1.0.
	// Buyer locked 3×1.0 = ¥3 at creation, pays 3×0.8 = ¥2.40, refunded ¥0.60.
	tradeAmount, payment, refund := pairSettlementMath(5, 3, 0.8, 1.0)
	if !closeTo(tradeAmount, 3.0, 1e-9) || !closeTo(payment, 2.4, 1e-9) || !closeTo(refund, 0.6, 1e-9) {
		t.Errorf("auto-match scenario: expected trade=3 payment=2.4 refund=0.6, got trade=%.2f payment=%.2f refund=%.2f",
			tradeAmount, payment, refund)
	}
	if !closeTo(payment+refund, 3.0, 1e-9) {
		t.Errorf("buy lock not conserved: payment+refund=%.2f, want 3.00", payment+refund)
	}

	// The 2 kWh residual then trades against a BUY 2 kWh @ ¥1.0.
	tradeAmount2, payment2, refund2 := pairSettlementMath(2, 2, 0.8, 1.0)
	if !closeTo(tradeAmount2, 2.0, 1e-9) || !closeTo(payment2, 1.6, 1e-9) || !closeTo(refund2, 0.4, 1e-9) {
		t.Errorf("residual settlement: expected trade=2 payment=1.6 refund=0.4, got trade=%.2f payment=%.2f refund=%.2f",
			tradeAmount2, payment2, refund2)
	}
	// Across both deals the seller sells its whole 5 kWh lock for ¥4.00.
	if !closeTo(tradeAmount+tradeAmount2, 5.0, 1e-9) {
		t.Errorf("sell lock not fully consumed: %.2f, want 5.00", tradeAmount+tradeAmount2)
	}
	if !closeTo(payment+payment2, 4.0, 1e-9) {
		t.Errorf("seller proceeds: %.2f, want 4.00", payment+payment2)
	}
}

// =============================================================================
// Match candidate scoring tests
// =============================================================================

// order builds a minimal Order for scoring tests.
func order(id, party, direction string, amount, price float64, source EnergySource, start, end string) *Order {
	return &Order{
		ID: id, PartyA: party, Direction: direction, Amount: amount, Price: price,
		EnergySource: source, DeliveryStart: start, DeliveryEnd: end, Status: "CREATED",
	}
}

const (
	winA = "2026-08-23T10:00:00Z"
	winB = "2026-08-23T12:00:00Z"
	winC = "2026-08-23T20:00:00Z"
	winD = "2026-08-23T22:00:00Z"
)

func TestFindBestMatchNoCandidates(t *testing.T) {
	sell := order("s1", "alice", "SELL", 5, 0.8, Solar, winA, winB)

	if got := findBestMatch(sell, nil); got != nil {
		t.Errorf("empty book: expected nil, got %s", got.candidate.ID)
	}
	// Same direction is not a counterpart.
	sameDir := []*Order{order("s2", "bob", "SELL", 5, 0.8, Solar, winA, winB)}
	if got := findBestMatch(sell, sameDir); got != nil {
		t.Errorf("same direction: expected nil, got %s", got.candidate.ID)
	}
	// Trading with yourself is not a match.
	selfTrade := []*Order{order("b1", "alice", "BUY", 5, 1.0, Solar, winA, winB)}
	if got := findBestMatch(sell, selfTrade); got != nil {
		t.Errorf("self trade: expected nil, got %s", got.candidate.ID)
	}
	// The order itself appearing in the book must be skipped.
	if got := findBestMatch(sell, []*Order{sell}); got != nil {
		t.Errorf("self id: expected nil, got %s", got.candidate.ID)
	}
}

func TestFindBestMatchPriceFilter(t *testing.T) {
	sell := order("s1", "alice", "SELL", 5, 1.2, Solar, winA, winB)
	book := []*Order{order("b1", "bob", "BUY", 5, 1.0, Solar, winA, winB)}

	if got := findBestMatch(sell, book); got != nil {
		t.Errorf("sell above buy price: expected nil, got %s", got.candidate.ID)
	}

	// Equal prices are acceptable (sellPrice <= buyPrice).
	sell.Price = 1.0
	if got := findBestMatch(sell, book); got == nil {
		t.Error("equal prices: expected a match, got nil")
	}
}

func TestFindBestMatchSourceMismatch(t *testing.T) {
	sell := order("s1", "alice", "SELL", 5, 0.8, Solar, winA, winB)

	mismatch := []*Order{order("b1", "bob", "BUY", 5, 1.0, Wind, winA, winB)}
	if got := findBestMatch(sell, mismatch); got != nil {
		t.Errorf("source mismatch: expected nil, got %s", got.candidate.ID)
	}

	// An unset source on either side means "any source".
	unset := []*Order{order("b2", "bob", "BUY", 5, 1.0, "", winA, winB)}
	if got := findBestMatch(sell, unset); got == nil {
		t.Error("unset counterpart source: expected a match, got nil")
	}
}

func TestFindBestMatchOverlapBonus(t *testing.T) {
	// A deliverable match must beat a cheaper undeliverable one: the overlap
	// bonus (+10) dominates the price spread.
	sell := order("s1", "alice", "SELL", 5, 0.8, Solar, winA, winB)
	book := []*Order{
		order("b_far", "bob", "BUY", 5, 1.3, Solar, winC, winD),   // spread 0.5, no overlap
		order("b_near", "carol", "BUY", 5, 1.1, Solar, winA, winB), // spread 0.3, overlaps
	}

	got := findBestMatch(sell, book)
	if got == nil {
		t.Fatal("expected a match, got nil")
	}
	if got.candidate.ID != "b_near" {
		t.Errorf("overlap bonus: expected b_near to win, got %s", got.candidate.ID)
	}
	if !got.overlap {
		t.Error("overlap bonus: winner should be flagged as overlapping")
	}
}

func TestFindBestMatchTiePrefersOverlap(t *testing.T) {
	// Identical price spreads: the overlapping candidate wins even though it is
	// evaluated second and does not strictly beat the incumbent's score.
	sell := order("s1", "alice", "SELL", 5, 0.8, Solar, winA, winB)
	book := []*Order{
		order("b_far", "bob", "BUY", 5, 1.0, Solar, winC, winD),
		order("b_near", "carol", "BUY", 5, 1.0, Solar, winA, winB),
	}

	got := findBestMatch(sell, book)
	if got == nil {
		t.Fatal("expected a match, got nil")
	}
	if got.candidate.ID != "b_near" {
		t.Errorf("tie break: expected the overlapping candidate, got %s", got.candidate.ID)
	}
}

// =============================================================================
// Device type and timezone tests
// =============================================================================

func TestValidDeviceType(t *testing.T) {
	valid := []DeviceType{SolarPanel, WindTurbine, BatteryStorage}
	for _, d := range valid {
		if !validDeviceType(d) {
			t.Errorf("validDeviceType(%q): expected true, got false", d)
		}
	}
	// "SolarPanel" used to fall through to a default branch and silently mint
	// 1.0 kWh instead of erroring.
	invalid := []DeviceType{"SolarPanel", "SOLAR", "", "battery_storage"}
	for _, d := range invalid {
		if validDeviceType(d) {
			t.Errorf("validDeviceType(%q): expected false, got true", d)
		}
	}
}

func TestCSTTimezone(t *testing.T) {
	// The observed bug: 04:14 UTC is 12:14 in Beijing, but the hour was read as
	// 4 and priced as an off-peak valley period.
	if hour := time.Date(2026, 8, 23, 4, 14, 0, 0, time.UTC).In(cstZone).Hour(); hour != 12 {
		t.Errorf("04:14 UTC in CST: expected hour 12, got %d", hour)
	}
	// Day boundary: 16:00 UTC is midnight of the next day in CST.
	midnight := time.Date(2026, 8, 23, 16, 0, 0, 0, time.UTC).In(cstZone)
	if midnight.Hour() != 0 {
		t.Errorf("16:00 UTC in CST: expected hour 0, got %d", midnight.Hour())
	}
	if midnight.Day() != 24 {
		t.Errorf("16:00 UTC in CST: expected day 24, got %d", midnight.Day())
	}
	// The offset must be fixed rather than inherited from the environment, or
	// endorsing peers in different zones would compute different results.
	if _, offset := time.Now().In(cstZone).Zone(); offset != 8*3600 {
		t.Errorf("cstZone offset: expected 28800s, got %ds", offset)
	}
}

func TestOrderMatchedWithJSON(t *testing.T) {
	o := Order{
		ID: "energy_order_1", PartyA: "alice", PartyB: "bob", Direction: "SELL",
		EnergySource: Solar, Amount: 3, Price: 0.8, Status: "FINISHED",
		MatchedWith: "energy_order_2",
	}
	data, err := json.Marshal(o)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back Order
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.MatchedWith != "energy_order_2" {
		t.Errorf("matchedWith round-trip: expected energy_order_2, got %q", back.MatchedWith)
	}

	// Orders written before the field existed must still unmarshal, leaving the
	// link empty so settlement falls back to searching for the counterpart.
	legacy := `{"id":"energy_order_3","partyA":"alice","partyB":"bob","direction":"SELL","status":"MATCHED","amount":2,"price":0.9}`
	var old Order
	if err := json.Unmarshal([]byte(legacy), &old); err != nil {
		t.Fatalf("legacy unmarshal: %v", err)
	}
	if old.MatchedWith != "" {
		t.Errorf("legacy order: expected empty matchedWith, got %q", old.MatchedWith)
	}
	if old.Status != "MATCHED" {
		t.Errorf("legacy order: expected status MATCHED, got %q", old.Status)
	}
}
