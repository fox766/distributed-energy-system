package chaincode

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// SmartContract provides the energy trading business logic
type SmartContract struct {
	contractapi.Contract
}

// Init initializes the energy market parameters on the ledger.
// Does NOT overwrite if already initialized.
func (s *SmartContract) Init(ctx contractapi.TransactionContextInterface) error {
	// Check if already initialized — prevent data loss on restart
	exists, err := s.ItemExists(ctx, EnergyStatusKey)
	if err != nil {
		return fmt.Errorf("failed to check EnergyStatus: %v", err)
	}
	if exists {
		// Already initialized — ensure TOU schedule exists, then return
		touExists, _ := s.ItemExists(ctx, TOUScheduleKey)
		if !touExists {
			s.InitTOUSchedule(ctx)
		}
		return nil
	}

	status := EnergyStatus{
		EnergyPrice: 1.0,
		Fee:         0.01,
	}
	statusJSON, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal EnergyStatus: %v", err)
	}
	if err := ctx.GetStub().PutState(EnergyStatusKey, statusJSON); err != nil {
		return fmt.Errorf("failed to save EnergyStatus: %v", err)
	}
	// Initialize global counters
	if err := ctx.GetStub().PutState(UserCountKey, []byte("0")); err != nil {
		return err
	}
	if err := ctx.GetStub().PutState(OrderCountKey, []byte("0")); err != nil {
		return err
	}
	if err := ctx.GetStub().PutState(PriceHistoryCountKey, []byte("0")); err != nil {
		return err
	}
	if err := ctx.GetStub().PutState(CarbonCountKey, []byte("0")); err != nil {
		return err
	}

	// Initialize TOU schedule
	if err := s.InitTOUSchedule(ctx); err != nil {
		return err
	}

	// Emit initialization event
	ctx.GetStub().SetEvent("EnergySystemInit", statusJSON)
	return nil
}

// =============================================================================
// User Management
// =============================================================================

// RegisterUser creates a new user on the ledger.
func (s *SmartContract) RegisterUser(ctx contractapi.TransactionContextInterface,
	userid, username, userrole string, available, balance float64) error {

	exists, err := s.ItemExists(ctx, userid)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("user %s already exists", userid)
	}

	user := User{
		UserID:    userid,
		UserName:  username,
		UserRole:  userrole,
		Available: available,
		Balance:   balance,
	}
	userJSON, err := json.Marshal(user)
	if err != nil {
		return err
	}
	if err := ctx.GetStub().PutState(userid, userJSON); err != nil {
		return err
	}

	// Increment user count
	s.incrementCounter(ctx, UserCountKey)

	s.recordAudit(ctx, "RegisterUser", "", userid,
		fmt.Sprintf("user=%s role=%s", username, userrole))

	ctx.GetStub().SetEvent("UserRegistered", userJSON)
	return nil
}

// GetUser retrieves a user by ID.
func (s *SmartContract) GetUser(ctx contractapi.TransactionContextInterface, id string) (*User, error) {
	exists, err := s.ItemExists(ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("user %s does not exist", id)
	}
	userJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read user from world state: %v", err)
	}
	var user User
	if err := json.Unmarshal(userJSON, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUser updates a user's available energy and/or balance.
// Non-negative values trigger an update; negative values leave the field unchanged.
func (s *SmartContract) UpdateUser(ctx contractapi.TransactionContextInterface,
	id string, available, balance float64) error {

	user, err := s.GetUser(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get user while updating: %v", err)
	}
	if available >= 0 {
		user.Available = available
	}
	if balance >= 0 {
		user.Balance = balance
	}
	userJSON, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user: %v", err)
	}
	if err := ctx.GetStub().PutState(id, userJSON); err != nil {
		return fmt.Errorf("failed to save user: %v", err)
	}
	return nil
}

// =============================================================================
// Order Management
// =============================================================================

// CreateOrder creates a new energy trade order on the ledger.
func (s *SmartContract) CreateOrder(ctx contractapi.TransactionContextInterface,
	id, partyA, direction, energySource, deliveryStart, deliveryEnd string,
	amount, price, fee float64) error {

	// Validate direction
	if direction != "SELL" && direction != "BUY" {
		return fmt.Errorf("direction must be SELL or BUY, got %s", direction)
	}

	// For SELL orders, verify the seller has enough available energy
	if direction == "SELL" {
		userA, err := s.GetUser(ctx, partyA)
		if err != nil {
			return fmt.Errorf("failed to find seller %s: %v", partyA, err)
		}
		if userA.Available < amount {
			return fmt.Errorf("insufficient available energy: have %.2f kWh, need %.2f kWh",
				userA.Available, amount)
		}
		userA.Available -= amount
		if err := s.saveUser(ctx, userA); err != nil {
			return err
		}
	}

	// For BUY orders, verify the buyer has enough balance
	if direction == "BUY" {
		userA, err := s.GetUser(ctx, partyA)
		if err != nil {
			return fmt.Errorf("failed to find buyer %s: %v", partyA, err)
		}
		totalCost := amount * price
		if userA.Balance < totalCost {
			return fmt.Errorf("insufficient balance: have ¥%.2f, need ¥%.2f",
				userA.Balance, totalCost)
		}
		userA.Balance -= totalCost
		if err := s.saveUser(ctx, userA); err != nil {
			return err
		}
	}

	order := Order{
		ID:            id,
		PartyA:        partyA,
		PartyB:        "",
		Direction:     direction,
		EnergySource:  EnergySource(energySource),
		Amount:        amount,
		Price:         price,
		Fee:           fee,
		Status:        "CREATED",
		DeliveryStart: deliveryStart,
		DeliveryEnd:   deliveryEnd,
		CreatedAt:     s.now(ctx),
	}

	orderJSON, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal order: %v", err)
	}

	// Store order by ID
	if err := ctx.GetStub().PutState(id, orderJSON); err != nil {
		return fmt.Errorf("failed to save order: %v", err)
	}

	// Composite key: order~status~{status}~{id}
	statusKey, err := ctx.GetStub().CreateCompositeKey(OrderStatusPrefix, []string{order.Status, id})
	if err != nil {
		return err
	}
	ctx.GetStub().PutState(statusKey, []byte{0})

	// Composite key: order~user~{userid}~{id}
	userKey, err := ctx.GetStub().CreateCompositeKey(OrderUserPrefix, []string{partyA, id})
	if err != nil {
		return err
	}
	ctx.GetStub().PutState(userKey, []byte{0})

	// Increment order count
	s.incrementCounter(ctx, OrderCountKey)

	s.recordAudit(ctx, "CreateOrder", id, partyA,
		fmt.Sprintf("dir=%s src=%s amt=%.0fkWh price=%.2f", direction, energySource, amount, price))

	ctx.GetStub().SetEvent("OrderCreated", orderJSON)
	return nil
}

// MatchOrder matches a CREATED order to a buyer (partyB).
// MatchOrder manually pairs two CREATED orders.
//
// The counterpart is named by ORDER id, not by user id. Identifying it by user
// left a matched order with no link back to the order it traded against, so
// each side of a pair could later be settled on its own terms — the same deal
// paid for twice, with energy conjured out of nothing. Both orders now record
// each other in MatchedWith, and settlement works on the pair.
func (s *SmartContract) MatchOrder(ctx contractapi.TransactionContextInterface,
	id, counterpartOrderID string) error {

	if id == counterpartOrderID {
		return fmt.Errorf("cannot match order %s with itself", id)
	}

	order, err := s.GetOrder(ctx, id)
	if err != nil {
		return fmt.Errorf("get order %s failed: %v", id, err)
	}
	counterpart, err := s.GetOrder(ctx, counterpartOrderID)
	if err != nil {
		return fmt.Errorf("get counterpart order %s failed: %v", counterpartOrderID, err)
	}
	if order.Status != "CREATED" {
		return fmt.Errorf("order %s has status %s, expected CREATED", id, order.Status)
	}
	if counterpart.Status != "CREATED" {
		return fmt.Errorf("counterpart order %s has status %s, expected CREATED",
			counterpartOrderID, counterpart.Status)
	}
	if order.PartyA == counterpart.PartyA {
		return fmt.Errorf("cannot match your own order")
	}
	if order.Direction == counterpart.Direction {
		return fmt.Errorf("orders must have opposite directions, both are %s", order.Direction)
	}
	if order.EnergySource != "" && counterpart.EnergySource != "" &&
		order.EnergySource != counterpart.EnergySource {
		return fmt.Errorf("energy source mismatch: %s vs %s",
			order.EnergySource, counterpart.EnergySource)
	}

	sell, buy := order, counterpart
	if order.Direction == "BUY" {
		sell, buy = counterpart, order
	}
	if sell.Price > buy.Price {
		return fmt.Errorf("sell price ¥%.2f exceeds buy price ¥%.2f", sell.Price, buy.Price)
	}

	// No balance check at match time: funds/energy were already locked when
	// the order was created (BUY: Balance, SELL: Available).

	order.PartyB = counterpart.PartyA
	order.MatchedWith = counterpart.ID
	counterpart.PartyB = order.PartyA
	counterpart.MatchedWith = order.ID

	for _, o := range []*Order{order, counterpart} {
		oldStatusKey, _ := ctx.GetStub().CreateCompositeKey(OrderStatusPrefix, []string{o.Status, o.ID})
		ctx.GetStub().DelState(oldStatusKey)

		o.Status = "MATCHED"
		orderJSON, err := json.Marshal(o)
		if err != nil {
			return fmt.Errorf("failed to marshal order: %v", err)
		}
		if err := ctx.GetStub().PutState(o.ID, orderJSON); err != nil {
			return fmt.Errorf("failed to save order: %v", err)
		}

		newStatusKey, _ := ctx.GetStub().CreateCompositeKey(OrderStatusPrefix, []string{o.Status, o.ID})
		ctx.GetStub().PutState(newStatusKey, []byte{0})

		// Index the order under its counterparty too
		userKey, _ := ctx.GetStub().CreateCompositeKey(OrderUserPrefix, []string{o.PartyB, o.ID})
		ctx.GetStub().PutState(userKey, []byte{0})
	}

	s.recordAudit(ctx, "MatchOrder", id, counterpart.PartyA,
		fmt.Sprintf("matched SELL:%s + BUY:%s @ ¥%.2f", sell.ID, buy.ID, sell.Price))

	orderJSON, _ := json.Marshal(order)
	ctx.GetStub().SetEvent("OrderMatched", orderJSON)
	return nil
}

// SettleOrder completes a MATCHED order by settling the whole pair.
//
// Settling one side alone was the double-spend: a pair is two orders, each
// individually settleable, so the same trade moved money and energy twice.
// Resolving the counterpart and running both through executePairSettlement
// means the deal is accounted exactly once and the second call finds both
// orders FINISHED and refuses.
func (s *SmartContract) SettleOrder(ctx contractapi.TransactionContextInterface, id string) error {
	order, err := s.GetOrder(ctx, id)
	if err != nil {
		return err
	}
	if order.Status != "MATCHED" {
		return fmt.Errorf("order %s has status %s, expected MATCHED", id, order.Status)
	}

	counterpart, err := s.resolveCounterpart(ctx, order)
	if err != nil {
		return err
	}

	sell, buy := order, counterpart
	if order.Direction == "BUY" {
		sell, buy = counterpart, order
	}

	_, err = s.executePairSettlement(ctx, sell, buy, "MATCHED")
	return err
}

// resolveCounterpart finds the order on the other side of a matched pair.
//
// Orders matched after the MatchedWith field was introduced carry the link
// directly. Orders already sitting in MATCHED on the ledger from before it do
// not, so fall back to searching the counterparty's orders for the unique
// opposite-direction MATCHED one.
func (s *SmartContract) resolveCounterpart(ctx contractapi.TransactionContextInterface,
	order *Order) (*Order, error) {

	if order.MatchedWith != "" {
		counterpart, err := s.GetOrder(ctx, order.MatchedWith)
		if err != nil {
			return nil, fmt.Errorf("counterpart order %s not found: %v", order.MatchedWith, err)
		}
		if counterpart.Status != "MATCHED" {
			return nil, fmt.Errorf("counterpart order %s has status %s, expected MATCHED",
				counterpart.ID, counterpart.Status)
		}
		return counterpart, nil
	}

	if order.PartyB == "" {
		return nil, fmt.Errorf("order %s is MATCHED but records no counterparty", order.ID)
	}

	candidates, err := s.GetUserOrders(ctx, order.PartyB)
	if err != nil {
		return nil, fmt.Errorf("failed to search counterpart orders: %v", err)
	}
	var found []*Order
	for _, c := range candidates {
		if c.ID == order.ID || c.Status != "MATCHED" ||
			c.Direction == order.Direction || c.PartyA != order.PartyB {
			continue
		}
		found = append(found, c)
	}
	switch len(found) {
	case 1:
		return found[0], nil
	case 0:
		return nil, fmt.Errorf("order %s is MATCHED but its counterpart order cannot be found; cancel and re-create it", order.ID)
	default:
		return nil, fmt.Errorf("order %s has %d possible counterpart orders; settle them via MatchOrder to disambiguate",
			order.ID, len(found))
	}
}

// CancelOrder cancels a CREATED order and returns locked resources.
func (s *SmartContract) CancelOrder(ctx contractapi.TransactionContextInterface, id string) error {
	order, err := s.GetOrder(ctx, id)
	if err != nil {
		return err
	}
	if order.Status != "CREATED" {
		return fmt.Errorf("only CREATED orders can be cancelled, got status %s", order.Status)
	}

	// Return locked resources to partyA
	userA, err := s.GetUser(ctx, order.PartyA)
	if err != nil {
		return err
	}

	if order.Direction == "SELL" {
		userA.Available += order.Amount
	} else {
		// BUY order: refund the locked balance
		userA.Balance += order.Amount * order.Price
	}

	if err := s.saveUser(ctx, userA); err != nil {
		return err
	}

	// Update composite keys
	oldStatusKey, _ := ctx.GetStub().CreateCompositeKey(OrderStatusPrefix, []string{order.Status, id})
	ctx.GetStub().DelState(oldStatusKey)

	order.Status = "CANCELLED"
	orderJSON, _ := json.Marshal(order)
	ctx.GetStub().PutState(id, orderJSON)

	// Index under the new status too. Without this the order is only removed
	// from the CREATED index and never added anywhere, so GetAllOrders on
	// "CANCELLED" (and hence "ALL") silently loses every cancelled order.
	newStatusKey, _ := ctx.GetStub().CreateCompositeKey(OrderStatusPrefix, []string{order.Status, id})
	ctx.GetStub().PutState(newStatusKey, []byte{0})

	s.recordAudit(ctx, "CancelOrder", id, userA.UserID,
		fmt.Sprintf("cancelled, returned %.0fkWh or ¥%.2f", order.Amount, order.Amount*order.Price))

	ctx.GetStub().SetEvent("OrderCancelled", orderJSON)
	return nil
}

// GetOrder retrieves an order by ID.
func (s *SmartContract) GetOrder(ctx contractapi.TransactionContextInterface, id string) (*Order, error) {
	exists, err := s.ItemExists(ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("order %s does not exist", id)
	}
	orderJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read order: %v", err)
	}
	var order Order
	if err := json.Unmarshal(orderJSON, &order); err != nil {
		return nil, err
	}
	return &order, nil
}

// GetAllOrders returns all orders, optionally filtered by status.
func (s *SmartContract) GetAllOrders(ctx contractapi.TransactionContextInterface, status string) ([]*Order, error) {
	var orders []*Order

	statuses := []string{status}
	if status == "" || status == "ALL" {
		statuses = []string{"CREATED", "MATCHED", "FINISHED", "CANCELLED"}
	}

	for _, st := range statuses {
		iter, err := ctx.GetStub().GetStateByPartialCompositeKey(OrderStatusPrefix, []string{st})
		if err != nil {
			continue
		}
		for iter.HasNext() {
			kv, err := iter.Next()
			if err != nil {
				continue
			}
			_, parts, err := ctx.GetStub().SplitCompositeKey(kv.Key)
			if err != nil || len(parts) < 2 {
				continue
			}
			orderID := parts[len(parts)-1]
			order, err := s.GetOrder(ctx, orderID)
			if err != nil {
				continue
			}
			orders = append(orders, order)
		}
		iter.Close()
	}

	sort.Slice(orders, func(i, j int) bool {
		return orders[i].CreatedAt.After(orders[j].CreatedAt)
	})
	return orders, nil
}

// GetUserOrders returns all orders where a given user is partyA or partyB.
func (s *SmartContract) GetUserOrders(ctx contractapi.TransactionContextInterface,
	userid string) ([]*Order, error) {

	iter, err := ctx.GetStub().GetStateByPartialCompositeKey(OrderUserPrefix, []string{userid})
	if err != nil {
		return nil, fmt.Errorf("failed to query orders for user %s: %v", userid, err)
	}
	defer iter.Close()

	var orders []*Order
	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			continue
		}
		_, parts, err := ctx.GetStub().SplitCompositeKey(kv.Key)
		if err != nil || len(parts) < 2 {
			continue
		}
		orderID := parts[len(parts)-1]
		order, err := s.GetOrder(ctx, orderID)
		if err != nil {
			continue
		}
		orders = append(orders, order)
	}

	sort.Slice(orders, func(i, j int) bool {
		return orders[i].CreatedAt.After(orders[j].CreatedAt)
	})
	return orders, nil
}

// =============================================================================
// Energy Market
// =============================================================================

// GetEnergyStatus returns the current energy market parameters.
func (s *SmartContract) GetEnergyStatus(ctx contractapi.TransactionContextInterface) (*EnergyStatus, error) {
	energyJSON, err := ctx.GetStub().GetState(EnergyStatusKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read energy status: %v", err)
	}
	var energy EnergyStatus
	if err := json.Unmarshal(energyJSON, &energy); err != nil {
		return nil, err
	}
	return &energy, nil
}

// UpdateEnergyPrice updates the energy price and records it in price history.
func (s *SmartContract) UpdateEnergyPrice(ctx contractapi.TransactionContextInterface,
	price, fee float64) error {

	newStatus := EnergyStatus{
		EnergyPrice: price,
		Fee:         fee,
	}
	newStatusJSON, err := json.Marshal(newStatus)
	if err != nil {
		return err
	}
	if err := ctx.GetStub().PutState(EnergyStatusKey, newStatusJSON); err != nil {
		return err
	}

	// Record in price history
	s.recordPriceHistory(ctx, price, 0)

	ctx.GetStub().SetEvent("EnergyPriceUpdated", newStatusJSON)
	return nil
}

// GetPriceHistory returns recent price records for charting.
func (s *SmartContract) GetPriceHistory(ctx contractapi.TransactionContextInterface, limit int) ([]*PriceRecord, error) {
	if limit <= 0 {
		limit = 7
	}
	var records []*PriceRecord

	iter, err := ctx.GetStub().GetStateByPartialCompositeKey(PriceHistoryPrefix, []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to query price history: %v", err)
	}
	defer iter.Close()

	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			return nil, err
		}
		var record PriceRecord
		if err := json.Unmarshal(kv.Value, &record); err != nil {
			continue
		}
		records = append(records, &record)
	}

	// Sort newest first
	sort.Slice(records, func(i, j int) bool {
		return records[i].Timestamp.After(records[j].Timestamp)
	})

	if len(records) > limit {
		records = records[:limit]
	}
	return records, nil
}

// getSystemCounts returns user count and order count from the ledger.
func (s *SmartContract) getSystemCounts(ctx contractapi.TransactionContextInterface) (int, int) {
	userCount, _ := s.getCounter(ctx, UserCountKey)
	orderCount, _ := s.getCounter(ctx, OrderCountKey)
	return userCount, orderCount
}

// GetSystemCounts returns user and order counts as a JSON string (exported for chaincode API).
func (s *SmartContract) GetSystemCounts(ctx contractapi.TransactionContextInterface) (string, error) {
	uc, oc := s.getSystemCounts(ctx)
	return fmt.Sprintf(`{"userCount":%d,"orderCount":%d}`, uc, oc), nil
}

// =============================================================================
// Carbon Tracking
// =============================================================================

// GetCarbonStats returns total carbon savings from all settled trades.
func (s *SmartContract) GetCarbonStats(ctx contractapi.TransactionContextInterface) (string, error) {
	var totalCarbon float64
	var totalTrades int

	iter, err := ctx.GetStub().GetStateByPartialCompositeKey(CarbonPrefix, []string{})
	if err != nil {
		return "", fmt.Errorf("failed to read carbon records: %v", err)
	}
	defer iter.Close()

	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			continue
		}
		var rec CarbonRecord
		if json.Unmarshal(kv.Value, &rec) == nil {
			totalCarbon += rec.CarbonSaved
			totalTrades++
		}
	}

	return fmt.Sprintf(`{"totalCarbonSaved":%.2f,"totalGreenTrades":%d}`, totalCarbon, totalTrades), nil
}

// GetCarbonHistory returns recent carbon records.
func (s *SmartContract) GetCarbonHistory(ctx contractapi.TransactionContextInterface, limit int) (string, error) {
	if limit <= 0 {
		limit = 20
	}
	var records []CarbonRecord
	iter, err := ctx.GetStub().GetStateByPartialCompositeKey(CarbonPrefix, []string{})
	if err != nil {
		return "[]", nil
	}
	defer iter.Close()
	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			continue
		}
		var rec CarbonRecord
		if json.Unmarshal(kv.Value, &rec) == nil {
			records = append(records, rec)
		}
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].Timestamp.After(records[j].Timestamp)
	})
	if len(records) > limit {
		records = records[:limit]
	}
	result, _ := json.Marshal(records)
	return string(result), nil
}

// recordCarbon stores a carbon savings record on settlement.
func (s *SmartContract) recordCarbon(ctx contractapi.TransactionContextInterface, order *Order) error {
	coeff := order.EnergySource.CarbonFactor()
	carbonSaved := order.Amount * coeff

	rec := CarbonRecord{
		OrderID:    order.ID,
		Amount:     order.Amount,
		Source:     string(order.EnergySource),
		Coefficient: coeff,
		CarbonSaved: carbonSaved,
		Timestamp:  s.now(ctx),
	}
	recJSON, _ := json.Marshal(rec)

	count, _ := s.incrementCounter(ctx, CarbonCountKey)
	key, err := ctx.GetStub().CreateCompositeKey(CarbonPrefix, []string{
		s.now(ctx).Format(time.RFC3339Nano),
		fmt.Sprintf("carbon_%d", count),
	})
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(key, recJSON)
}

// =============================================================================
// Audit Trail — immutable operation log
// =============================================================================

// GetAuditLog returns recent audit entries.
func (s *SmartContract) GetAuditLog(ctx contractapi.TransactionContextInterface, limit int) (string, error) {
	if limit <= 0 {
		limit = 50
	}
	var entries []AuditEntry
	iter, err := ctx.GetStub().GetStateByPartialCompositeKey(AuditPrefix, []string{})
	if err != nil {
		return "[]", nil
	}
	defer iter.Close()
	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			continue
		}
		var entry AuditEntry
		if json.Unmarshal(kv.Value, &entry) == nil {
			entries = append(entries, entry)
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})
	if len(entries) > limit {
		entries = entries[:limit]
	}
	result, _ := json.Marshal(entries)
	return string(result), nil
}

// recordAudit writes an immutable audit entry to the ledger.
func (s *SmartContract) recordAudit(ctx contractapi.TransactionContextInterface,
	operation, orderID, userID, details string) error {

	entry := AuditEntry{
		TxID:      ctx.GetStub().GetTxID(),
		Operation: operation,
		OrderID:   orderID,
		UserID:    userID,
		Details:   details,
		Timestamp: s.now(ctx),
	}
	entryJSON, _ := json.Marshal(entry)

	key, err := ctx.GetStub().CreateCompositeKey(AuditPrefix, []string{
		s.now(ctx).Format(time.RFC3339Nano),
		entry.TxID,
	})
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(key, entryJSON)
}

// =============================================================================
// Utility Functions
// =============================================================================

// cstZone is the business timezone (UTC+8) for every hour-of-day decision the
// contract makes: time-of-use pricing periods and the generation curves.
//
// It is a fixed offset rather than the process timezone on purpose. time.Unix
// renders in the peer container's local zone, which is UTC — so "peak hours
// 9-12" was being evaluated against UTC and midday in Beijing was priced as
// off-peak. Reading TZ from the environment would fix that only as long as
// every peer container agreed; a mismatch would make endorsing peers compute
// different results for the same transaction and break consensus. A constant
// compiled into the chaincode cannot diverge.
var cstZone = time.FixedZone("CST", 8*3600)

// now returns the transaction timestamp deterministically (same for all endorsing peers).
func (s *SmartContract) now(ctx contractapi.TransactionContextInterface) time.Time {
	ts, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return time.Unix(0, 0).In(cstZone)
	}
	return time.Unix(ts.Seconds, int64(ts.Nanos)).In(cstZone)
}

// ItemExists checks whether a key exists in the world state.
func (s *SmartContract) ItemExists(ctx contractapi.TransactionContextInterface, item string) (bool, error) {
	itemJSON, err := ctx.GetStub().GetState(item)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}
	return itemJSON != nil, nil
}

// saveUser marshals and writes a user to the world state.
func (s *SmartContract) saveUser(ctx contractapi.TransactionContextInterface, user *User) error {
	userJSON, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(user.UserID, userJSON)
}

// incrementCounter atomically increments a counter key.
func (s *SmartContract) incrementCounter(ctx contractapi.TransactionContextInterface, key string) (int, error) {
	val, err := ctx.GetStub().GetState(key)
	if err != nil {
		return 0, err
	}
	count := 0
	if val != nil {
		count, _ = strconv.Atoi(string(val))
	}
	count++
	return count, ctx.GetStub().PutState(key, []byte(strconv.Itoa(count)))
}

// getCounter reads a counter value.
func (s *SmartContract) getCounter(ctx contractapi.TransactionContextInterface, key string) (int, error) {
	val, err := ctx.GetStub().GetState(key)
	if err != nil {
		return 0, err
	}
	if val == nil {
		return 0, nil
	}
	return strconv.Atoi(string(val))
}

// recordPriceHistory stores a price record with composite key for time-based queries.
func (s *SmartContract) recordPriceHistory(ctx contractapi.TransactionContextInterface,
	price, volume float64) error {

	now := s.now(ctx)
	record := PriceRecord{
		Timestamp: now,
		Price:     price,
		Volume:    volume,
	}
	recordJSON, err := json.Marshal(record)
	if err != nil {
		return err
	}

	timestampStr := now.Format(time.RFC3339Nano)
	count, _ := s.incrementCounter(ctx, PriceHistoryCountKey)
	id := fmt.Sprintf("price_%d", count)

	key, err := ctx.GetStub().CreateCompositeKey(PriceHistoryPrefix, []string{timestampStr, id})
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(key, recordJSON)
}


// TopUpBalance credits a user's balance.
//
// Consumers register with no funds and a BUY order escrows its full cost at
// creation, so without this there is no way for a consumer to ever buy.
// Authorization lives in the backend's admin-only route: every request reaches
// the ledger under the same gateway identity, so the chaincode cannot tell an
// admin from anyone else.
func (s *SmartContract) TopUpBalance(ctx contractapi.TransactionContextInterface,
	userID string, amount float64) error {

	if amount <= 0 {
		return fmt.Errorf("top-up amount must be positive, got %.2f", amount)
	}

	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return err
	}
	user.Balance += amount
	if err := s.saveUser(ctx, user); err != nil {
		return err
	}

	s.recordAudit(ctx, "TopUpBalance", "", userID,
		fmt.Sprintf("credited ¥%.2f, new balance ¥%.2f", amount, user.Balance))

	ctx.GetStub().SetEvent("BalanceTopUp", []byte(fmt.Sprintf(
		`{"userid":"%s","amount":%.2f,"balance":%.2f}`, userID, amount, user.Balance)))
	return nil
}

// =============================================================================
// P0: Time-of-Use Pricing
// =============================================================================

// InitTOUSchedule initializes the time-of-use pricing schedule on the ledger.
func (s *SmartContract) InitTOUSchedule(ctx contractapi.TransactionContextInterface) error {
	schedule := TimeOfUseSchedule{
		Periods: []TimeOfUsePeriod{
			{Name: "peak", StartHour: 9, EndHour: 12, Price: 1.2},
			{Name: "peak", StartHour: 17, EndHour: 21, Price: 1.2},
			{Name: "flat", StartHour: 6, EndHour: 9, Price: 0.8},
			{Name: "flat", StartHour: 12, EndHour: 17, Price: 0.8},
			{Name: "flat", StartHour: 21, EndHour: 23, Price: 0.8},
			{Name: "valley", StartHour: 0, EndHour: 6, Price: 0.4},
			{Name: "valley", StartHour: 23, EndHour: 24, Price: 0.4},
		},
		UpdatedAt: s.now(ctx),
	}
	scheduleJSON, err := json.Marshal(schedule)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(TOUScheduleKey, scheduleJSON)
}

// GetTimeOfUsePrice returns the current electricity price based on time of day.
func (s *SmartContract) GetTimeOfUsePrice(ctx contractapi.TransactionContextInterface) (string, error) {
	scheduleJSON, err := ctx.GetStub().GetState(TOUScheduleKey)
	if err != nil || scheduleJSON == nil {
		// Fallback: initialize TOU schedule
		if err := s.InitTOUSchedule(ctx); err != nil {
			return `{"price":1.0,"period":"default"}`, nil
		}
		scheduleJSON, _ = ctx.GetStub().GetState(TOUScheduleKey)
	}

	var schedule TimeOfUseSchedule
	if err := json.Unmarshal(scheduleJSON, &schedule); err != nil {
		return `{"price":1.0,"period":"default"}`, nil
	}

	now := s.now(ctx)
	hour := now.Hour()

	for _, p := range schedule.Periods {
		if hour >= p.StartHour && hour < p.EndHour {
			return fmt.Sprintf(`{"price":%.2f,"period":"%s","hour":%d}`, p.Price, p.Name, hour), nil
		}
	}

	return fmt.Sprintf(`{"price":1.0,"period":"default","hour":%d}`, hour), nil
}

// =============================================================================
// P0: Auto Energy Generation (Virtual Smart Meter)
// =============================================================================

// GenerateEnergy simulates energy production based on device type and time of day.
// Solar: peaks at noon (12:00), zero at night
// Wind: random-ish based on hour, always some baseline
// Storage: constant discharge rate
func (s *SmartContract) GenerateEnergy(ctx contractapi.TransactionContextInterface,
	userID, deviceType string) error {

	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %v", err)
	}
	if user.UserRole != "PRODUCER" && user.UserRole != "admin" {
		return fmt.Errorf("user %s is not a producer", userID)
	}
	if !validDeviceType(DeviceType(deviceType)) {
		return fmt.Errorf("invalid deviceType %q: must be one of SOLAR_PANEL, WIND_TURBINE, BATTERY_STORAGE",
			deviceType)
	}

	now := s.now(ctx)
	hour := now.Hour()
	var generated float64

	switch DeviceType(deviceType) {
	case SolarPanel:
		// Solar: bell curve peaking at noon, 0 at night (20:00-5:00)
		if hour >= 5 && hour < 20 {
			// Peak at hour 12: sin curve approximation
			peakHour := 12.0
			dist := float64(hour) - peakHour
			generated = 5.0 * (1.0 - (dist*dist)/64.0) // max ~5 kWh/h at noon
			if generated < 0 {
				generated = 0
			}
		}
	case WindTurbine:
		// Wind: variable, higher at night and early morning
		baseOutput := 2.0
		// Add pseudo-randomness based on hour (deterministic from time)
		windFactor := 1.0 + float64((hour*7+int(now.Minute())/10*3)%10)/10.0
		generated = baseOutput * windFactor // 2-4 kWh/h
	case BatteryStorage:
		// Storage: constant discharge
		generated = 1.5 // 1.5 kWh/h steady
	}

	// Cap minimum generation
	if generated < 0.1 {
		generated = 0.1
	}

	user.Available += generated
	if err := s.saveUser(ctx, user); err != nil {
		return err
	}

	// Record generation event
	rec := GenerationRecord{
		UserID:    userID,
		DeviceType: DeviceType(deviceType),
		Amount:    generated,
		Timestamp: now,
	}
	recJSON, _ := json.Marshal(rec)

	count, _ := s.incrementCounter(ctx, GenerationCountKey)
	key, err := ctx.GetStub().CreateCompositeKey(GenerationPrefix, []string{
		userID,
		now.Format(time.RFC3339Nano),
		fmt.Sprintf("gen_%d", count),
	})
	if err != nil {
		return err
	}

	s.recordAudit(ctx, "GenerateEnergy", "", userID,
		fmt.Sprintf("device=%s generated=%.2fkWh available=%.2f", deviceType, generated, user.Available))

	return ctx.GetStub().PutState(key, recJSON)
}

// GetGenerationHistory returns generation records, optionally filtered by user.
func (s *SmartContract) GetGenerationHistory(ctx contractapi.TransactionContextInterface,
	userID string, limit int) (string, error) {
	if limit <= 0 {
		limit = 50
	}
	var records []GenerationRecord

	iter, err := ctx.GetStub().GetStateByPartialCompositeKey(GenerationPrefix, []string{})
	if err != nil {
		return "[]", nil
	}
	defer iter.Close()

	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			continue
		}
		var rec GenerationRecord
		if json.Unmarshal(kv.Value, &rec) != nil {
			continue
		}
		if userID != "" && rec.UserID != userID {
			continue
		}
		records = append(records, rec)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].Timestamp.After(records[j].Timestamp)
	})
	if len(records) > limit {
		records = records[:limit]
	}

	result, _ := json.Marshal(records)
	return string(result), nil
}

// =============================================================================
// P2: Auto-Matching Engine
// =============================================================================

// AutoMatchOrder tries to match a newly created order with existing opposite orders.
// Supports delivery window overlap checking and partial fills.
func (s *SmartContract) AutoMatchOrder(ctx contractapi.TransactionContextInterface,
	orderID string) (string, error) {

	order, err := s.GetOrder(ctx, orderID)
	if err != nil {
		return "", fmt.Errorf("order not found: %v", err)
	}
	if order.Status != "CREATED" {
		return "", fmt.Errorf("order %s is not in CREATED status", orderID)
	}

	allOrders, err := s.GetAllOrders(ctx, "CREATED")
	if err != nil {
		return "", err
	}

	best := findBestMatch(order, allOrders)
	if best == nil {
		return `{"matched":false,"message":"no compatible order found"}`, nil
	}

	// Determine SELL and BUY
	var sellOrder, buyOrder *Order
	if order.Direction == "SELL" {
		sellOrder = order
		buyOrder = best.candidate
	} else {
		sellOrder = best.candidate
		buyOrder = order
	}

	result, tradeAmount, residualID, err := s.tradePair(ctx, sellOrder, buyOrder)
	if err != nil {
		return "", err
	}

	s.recordAudit(ctx, "AutoMatch", orderID, "",
		fmt.Sprintf("auto-matched SELL:%s + BUY:%s amount=%.0fkWh @ ¥%.2f delivery_overlap=%v partial=%v",
			sellOrder.ID, buyOrder.ID, tradeAmount, sellOrder.Price, best.overlap, residualID != ""))

	return result, nil
}

// matchScore ranks a candidate counterpart order.
type matchScore struct {
	candidate *Order
	score     float64
	overlap   bool
}

// findBestMatch picks the best counterpart for order out of candidates, or nil
// when nothing is compatible.
//
// Compatible means: opposite direction, a different party, the same energy
// source when both orders name one, and a sell price the buyer is willing to
// pay. Among those, the widest price spread wins, with a heavy bonus for
// overlapping delivery windows so a deliverable match always beats a cheaper
// undeliverable one; ties go to the overlapping candidate.
//
// Pure function over already-loaded orders — no ledger access, so it is
// directly unit-testable.
func findBestMatch(order *Order, candidates []*Order) *matchScore {
	oppositeDir := "BUY"
	if order.Direction == "BUY" {
		oppositeDir = "SELL"
	}

	var best *matchScore
	for _, candidate := range candidates {
		if candidate.ID == order.ID {
			continue
		}
		if candidate.Direction != oppositeDir {
			continue
		}
		if candidate.PartyA == order.PartyA {
			continue
		}
		if order.EnergySource != "" && candidate.EnergySource != "" &&
			order.EnergySource != candidate.EnergySource {
			continue
		}

		// Price compatibility
		var sellPrice, buyPrice float64
		if oppositeDir == "SELL" {
			sellPrice, buyPrice = candidate.Price, order.Price
		} else {
			sellPrice, buyPrice = order.Price, candidate.Price
		}
		if sellPrice > buyPrice {
			continue
		}

		overlap := deliveryWindowsOverlap(order, candidate)
		score := buyPrice - sellPrice
		if overlap {
			score += 10.0 // heavy bonus for overlapping delivery windows
		}

		if best == nil || score > best.score ||
			(score == best.score && !best.overlap && overlap) {
			best = &matchScore{candidate: candidate, score: score, overlap: overlap}
		}
	}
	return best
}

// tradePair trades the overlap of a compatible SELL/BUY pair: it trims the
// larger order to the traded amount, carries the remainder into a residual
// order, and settles the pair. Returns the settlement JSON, the traded amount,
// and the residual order ID ("" when the amounts matched exactly).
//
// The residual is created without taking a second lock — the original order
// creation already locked the full amount, so re-locking the remainder would
// double-charge the party.
func (s *SmartContract) tradePair(ctx contractapi.TransactionContextInterface,
	sellOrder, buyOrder *Order) (string, float64, string, error) {

	tradeAmount := sellOrder.Amount
	if buyOrder.Amount < tradeAmount {
		tradeAmount = buyOrder.Amount
	}

	var residualID string
	switch {
	case sellOrder.Amount > buyOrder.Amount:
		residualID = sellOrder.ID + "_r"
		residualAmount := sellOrder.Amount - tradeAmount
		if err := s.updateOrderAmount(ctx, sellOrder, tradeAmount); err != nil {
			return "", 0, "", fmt.Errorf("partial fill: failed to trim sell order: %v", err)
		}
		if err := s.createResidualOrder(ctx, residualID, sellOrder.PartyA, "SELL",
			string(sellOrder.EnergySource), sellOrder.DeliveryStart, sellOrder.DeliveryEnd,
			residualAmount, sellOrder.Price, sellOrder.Fee); err != nil {
			return "", 0, "", fmt.Errorf("partial fill: failed to create residual sell order: %v", err)
		}
	case buyOrder.Amount > sellOrder.Amount:
		residualID = buyOrder.ID + "_r"
		residualAmount := buyOrder.Amount - tradeAmount
		if err := s.updateOrderAmount(ctx, buyOrder, tradeAmount); err != nil {
			return "", 0, "", fmt.Errorf("partial fill: failed to trim buy order: %v", err)
		}
		if err := s.createResidualOrder(ctx, residualID, buyOrder.PartyA, "BUY",
			string(buyOrder.EnergySource), buyOrder.DeliveryStart, buyOrder.DeliveryEnd,
			residualAmount, buyOrder.Price, buyOrder.Fee); err != nil {
			return "", 0, "", fmt.Errorf("partial fill: failed to create residual buy order: %v", err)
		}
	}

	result, err := s.settleMatchedPair(ctx, sellOrder, buyOrder)
	if err != nil {
		return "", 0, "", err
	}
	if residualID != "" {
		result = result[:len(result)-1] +
			fmt.Sprintf(`,"partial":true,"residualOrder":"%s","tradedAmount":%.0f}`, residualID, tradeAmount)
	}
	return result, tradeAmount, residualID, nil
}

// deliveryWindowsOverlap checks if two orders' delivery windows overlap.
func deliveryWindowsOverlap(a, b *Order) bool {
	if a.DeliveryStart == "" || a.DeliveryEnd == "" ||
		b.DeliveryStart == "" || b.DeliveryEnd == "" {
		return false
	}
	// Simple string comparison works for ISO 8601 datetime strings
	return a.DeliveryStart < b.DeliveryEnd && b.DeliveryStart < a.DeliveryEnd
}

// settleMatchedPair settles a matched order pair with a single transfer.
//
// Accounting model:
//   - BUY creation locked buyer.Balance -= buyAmount × buyPrice;
//     SELL creation locked seller.Available -= sellAmount.
//   - The deal trades tradeAmount at the SELL price: seller receives the
//     payment, buyer receives the energy.
//   - The buyer's locked spread (buy.Price - sell.Price) × tradeAmount is
//     refunded; any remainder of the locks stays attached to the residual
//     orders, which are created without a second deduction.
func (s *SmartContract) settleMatchedPair(ctx contractapi.TransactionContextInterface,
	sellOrder, buyOrder *Order) (string, error) {
	return s.executePairSettlement(ctx, sellOrder, buyOrder, "CREATED")
}

// executePairSettlement settles a SELL/BUY pair as a single deal.
//
// This is the only place money and energy move for a trade, which is what makes
// double settlement impossible: both orders reach FINISHED in one transaction,
// so settling either one again fails the status check. expectedStatus is
// "CREATED" when the auto-matcher settles on the spot, and "MATCHED" when a
// manually matched pair is settled later.
func (s *SmartContract) executePairSettlement(ctx contractapi.TransactionContextInterface,
	sellOrder, buyOrder *Order, expectedStatus string) (string, error) {

	if sellOrder.ID == buyOrder.ID {
		return "", fmt.Errorf("cannot settle order %s against itself", sellOrder.ID)
	}

	// Re-read both orders: guard against concurrent matching
	sell, err := s.GetOrder(ctx, sellOrder.ID)
	if err != nil {
		return "", fmt.Errorf("settle pair: get sell order failed: %v", err)
	}
	buy, err := s.GetOrder(ctx, buyOrder.ID)
	if err != nil {
		return "", fmt.Errorf("settle pair: get buy order failed: %v", err)
	}
	if sell.Status != expectedStatus || buy.Status != expectedStatus {
		return "", fmt.Errorf("settle pair: expected both orders %s, got sell=%s buy=%s",
			expectedStatus, sell.Status, buy.Status)
	}

	seller, err := s.GetUser(ctx, sell.PartyA)
	if err != nil {
		return "", err
	}
	buyer, err := s.GetUser(ctx, buy.PartyA)
	if err != nil {
		return "", err
	}

	tradeAmount, payment, refund := pairSettlementMath(sell.Amount, buy.Amount, sell.Price, buy.Price)

	seller.Balance += payment
	buyer.Balance += refund
	buyer.Available += tradeAmount

	if err := s.saveUser(ctx, seller); err != nil {
		return "", err
	}
	if err := s.saveUser(ctx, buyer); err != nil {
		return "", err
	}

	// Finish both orders in this transaction (single settlement, no second
	// money/energy movement)
	sell.PartyB = buy.PartyA
	buy.PartyB = sell.PartyA
	sell.MatchedWith = buy.ID
	buy.MatchedWith = sell.ID
	for _, order := range []*Order{sell, buy} {
		oldStatusKey, _ := ctx.GetStub().CreateCompositeKey(OrderStatusPrefix, []string{order.Status, order.ID})
		ctx.GetStub().DelState(oldStatusKey)
		order.Status = "FINISHED"
		orderJSON, err := json.Marshal(order)
		if err != nil {
			return "", err
		}
		if err := ctx.GetStub().PutState(order.ID, orderJSON); err != nil {
			return "", err
		}
		newStatusKey, _ := ctx.GetStub().CreateCompositeKey(OrderStatusPrefix, []string{order.Status, order.ID})
		ctx.GetStub().PutState(newStatusKey, []byte{0})
	}

	// OrderUser composite keys for the counterparties
	userKey, _ := ctx.GetStub().CreateCompositeKey(OrderUserPrefix, []string{buy.PartyA, sell.ID})
	ctx.GetStub().PutState(userKey, []byte{0})
	userKey2, _ := ctx.GetStub().CreateCompositeKey(OrderUserPrefix, []string{sell.PartyA, buy.ID})
	ctx.GetStub().PutState(userKey2, []byte{0})

	// One price / carbon / audit record for the deal
	if err := s.recordPriceHistory(ctx, sell.Price, tradeAmount); err != nil {
		return "", err
	}
	if err := s.recordCarbon(ctx, sell); err != nil {
		return "", err
	}

	s.recordAudit(ctx, "SettleOrder", sell.ID, buy.PartyA,
		fmt.Sprintf("pair-settled %.0fkWh @ ¥%.2f, carbon saved: %.2f kg",
			tradeAmount, sell.Price, tradeAmount*sell.EnergySource.CarbonFactor()))
	s.recordAudit(ctx, "SettleOrder", buy.ID, sell.PartyA,
		fmt.Sprintf("pair-settled %.0fkWh @ ¥%.2f, refund ¥%.2f",
			tradeAmount, sell.Price, refund))

	ctx.GetStub().SetEvent("OrderSettled", []byte(fmt.Sprintf(
		`{"sellOrder":"%s","buyOrder":"%s","price":%.2f,"amount":%.0f}`,
		sell.ID, buy.ID, sell.Price, tradeAmount)))

	return fmt.Sprintf(`{"matched":true,"sellOrder":"%s","buyOrder":"%s","price":%.2f,"amount":%.0f}`,
		sell.ID, buy.ID, sell.Price, tradeAmount), nil
}

// pairSettlementMath computes the single-transfer settlement for a matched
// order pair. Returns the traded amount, the payment at the SELL price, and
// the refund of the buyer's locked price spread (locked at BUY creation but
// never spent on the deal).
func pairSettlementMath(sellAmount, buyAmount, sellPrice, buyPrice float64) (tradeAmount, payment, refund float64) {
	tradeAmount = sellAmount
	if buyAmount < tradeAmount {
		tradeAmount = buyAmount
	}
	payment = tradeAmount * sellPrice
	refund = tradeAmount * (buyPrice - sellPrice)
	if refund < 0 {
		refund = 0 // defensive: the price filter requires sellPrice <= buyPrice
	}
	return
}

// createResidualOrder writes a residual order without touching user balances:
// the lock was already taken when the original order was created, and the
// residual represents the remainder of that same lock.
func (s *SmartContract) createResidualOrder(ctx contractapi.TransactionContextInterface,
	id, partyA, direction, energySource, deliveryStart, deliveryEnd string,
	amount, price, fee float64) error {

	order := Order{
		ID:            id,
		PartyA:        partyA,
		PartyB:        "",
		Direction:     direction,
		EnergySource:  EnergySource(energySource),
		Amount:        amount,
		Price:         price,
		Fee:           fee,
		Status:        "CREATED",
		DeliveryStart: deliveryStart,
		DeliveryEnd:   deliveryEnd,
		CreatedAt:     s.now(ctx),
	}

	orderJSON, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal order: %v", err)
	}
	if err := ctx.GetStub().PutState(id, orderJSON); err != nil {
		return fmt.Errorf("failed to save order: %v", err)
	}

	statusKey, err := ctx.GetStub().CreateCompositeKey(OrderStatusPrefix, []string{order.Status, id})
	if err != nil {
		return err
	}
	ctx.GetStub().PutState(statusKey, []byte{0})

	userKey, err := ctx.GetStub().CreateCompositeKey(OrderUserPrefix, []string{partyA, id})
	if err != nil {
		return err
	}
	ctx.GetStub().PutState(userKey, []byte{0})

	s.incrementCounter(ctx, OrderCountKey)

	s.recordAudit(ctx, "CreateResidualOrder", id, partyA,
		fmt.Sprintf("residual of partial fill: dir=%s src=%s amt=%.0fkWh price=%.2f",
			direction, energySource, amount, price))

	ctx.GetStub().SetEvent("OrderCreated", orderJSON)
	return nil
}

// updateOrderAmount trims an order to a new amount and persists it.
func (s *SmartContract) updateOrderAmount(ctx contractapi.TransactionContextInterface,
	order *Order, amount float64) error {
	order.Amount = amount
	orderJSON, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal order: %v", err)
	}
	return ctx.GetStub().PutState(order.ID, orderJSON)
}

// RunAutoMatch settles the single best matchable pair in the order book and
// reports whether it found one, as {"matched":0|1,...}.
//
// It deliberately stops after ONE pair. Fabric gives a transaction no
// read-your-writes: GetState returns committed state, not what this same
// transaction has already written. A loop settling a second pair here would
// therefore re-read the orders it just modified in their pre-transaction form
// and settle against stale amounts and balances. Callers drain the book by
// invoking this repeatedly — one transaction per pair — which is what
// handler.RunAutoMatchLoop in the backend does.
func (s *SmartContract) RunAutoMatch(ctx contractapi.TransactionContextInterface) (string, error) {
	allOrders, err := s.GetAllOrders(ctx, "CREATED")
	if err != nil {
		return "", err
	}

	sellOrders := make([]*Order, 0)
	buyOrders := make([]*Order, 0)
	for _, o := range allOrders {
		if o.Direction == "SELL" {
			sellOrders = append(sellOrders, o)
		} else {
			buyOrders = append(buyOrders, o)
		}
	}

	// Pick the globally best pair rather than the first workable one, so the
	// ordering of the state iterator does not decide who trades.
	var bestSell, bestBuy *Order
	var bestScore float64
	var bestOverlap bool
	for _, sell := range sellOrders {
		m := findBestMatch(sell, buyOrders)
		if m == nil {
			continue
		}
		if bestSell == nil || m.score > bestScore {
			bestSell, bestBuy, bestScore, bestOverlap = sell, m.candidate, m.score, m.overlap
		}
	}

	if bestSell == nil {
		return fmt.Sprintf(`{"matched":0,"remaining":%d}`, len(allOrders)), nil
	}

	_, tradeAmount, residualID, err := s.tradePair(ctx, bestSell, bestBuy)
	if err != nil {
		return "", err
	}

	s.recordAudit(ctx, "AutoMatch", bestSell.ID, "",
		fmt.Sprintf("batch-matched SELL:%s + BUY:%s amount=%.0fkWh @ ¥%.2f delivery_overlap=%v partial=%v",
			bestSell.ID, bestBuy.ID, tradeAmount, bestSell.Price, bestOverlap, residualID != ""))

	// settleMatchedPair's JSON carries "matched":true; re-render the pair
	// fields with the batch drain shape (matched as the pair count) so the
	// response never contains the key twice.
	partialField := ""
	if residualID != "" {
		partialField = fmt.Sprintf(`,"partial":true,"residualOrder":"%s","tradedAmount":%.0f`, residualID, tradeAmount)
	}
	return fmt.Sprintf(
		`{"matched":1,"remaining":%d,"sellOrder":"%s","buyOrder":"%s","price":%.2f,"amount":%.0f%s}`,
		len(allOrders)-2, bestSell.ID, bestBuy.ID, bestSell.Price, tradeAmount, partialField), nil
}

// =============================================================================
// P3: Transaction History / Statement
// =============================================================================

// GetTransactionHistory returns all FINISHED orders for a user (or all users if empty).
func (s *SmartContract) GetTransactionHistory(ctx contractapi.TransactionContextInterface,
	userID, month string, limit int) (string, error) {
	if limit <= 0 {
		limit = 100
	}

	allOrders, err := s.GetAllOrders(ctx, "FINISHED")
	if err != nil {
		return "[]", nil
	}

	var filtered []*Order
	for _, o := range allOrders {
		if userID != "" && o.PartyA != userID && o.PartyB != userID {
			continue
		}
		if month != "" {
			orderMonth := o.CreatedAt.Format("2006-01")
			if orderMonth != month {
				continue
			}
		}
		filtered = append(filtered, o)
	}

	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	result, _ := json.Marshal(filtered)
	return string(result), nil
}

// GetMonthlySummary returns a monthly statement for a user.
func (s *SmartContract) GetMonthlySummary(ctx contractapi.TransactionContextInterface,
	userID, month string) (string, error) {

	allOrders, err := s.GetAllOrders(ctx, "FINISHED")
	if err != nil {
		return "", err
	}

	summary := TransactionSummary{
		UserID: userID,
		Month:  month,
	}

	for _, o := range allOrders {
		orderMonth := o.CreatedAt.Format("2006-01")
		if month != "" && orderMonth != month {
			continue
		}
		if o.PartyA != userID && o.PartyB != userID {
			continue
		}

		if o.Direction == "SELL" {
			if o.PartyA == userID {
				// User was the seller
				summary.TotalIncome += o.Price * o.Amount
				summary.TotalSold += o.Amount
			} else {
				// User was the buyer of someone else's sell
				summary.TotalExpense += o.Price * o.Amount
				summary.TotalBought += o.Amount
			}
		} else {
			// BUY order
			if o.PartyA == userID {
				// User was the buyer
				summary.TotalExpense += o.Price * o.Amount
				summary.TotalBought += o.Amount
			} else {
				// User was the seller (matched to a BUY order)
				summary.TotalIncome += o.Price * o.Amount
				summary.TotalSold += o.Amount
			}
		}

		// Carbon saved: the energy amount * carbon factor
		coeff := o.EnergySource.CarbonFactor()
		summary.CarbonSaved += o.Amount * coeff
		summary.TradeCount++
	}

	result, _ := json.Marshal(summary)
	return string(result), nil
}

// GetUserStatement returns all monthly summaries for a user.
func (s *SmartContract) GetUserStatement(ctx contractapi.TransactionContextInterface,
	userID string) (string, error) {

	allOrders, err := s.GetAllOrders(ctx, "FINISHED")
	if err != nil {
		return "[]", nil
	}

	// Group by month
	monthMap := make(map[string]*TransactionSummary)
	for _, o := range allOrders {
		if o.PartyA != userID && o.PartyB != userID {
			continue
		}
		month := o.CreatedAt.Format("2006-01")
		ms, ok := monthMap[month]
		if !ok {
			ms = &TransactionSummary{UserID: userID, Month: month}
			monthMap[month] = ms
		}

		if o.Direction == "SELL" {
			if o.PartyA == userID {
				ms.TotalIncome += o.Price * o.Amount
				ms.TotalSold += o.Amount
			} else {
				ms.TotalExpense += o.Price * o.Amount
				ms.TotalBought += o.Amount
			}
		} else {
			if o.PartyA == userID {
				ms.TotalExpense += o.Price * o.Amount
				ms.TotalBought += o.Amount
			} else {
				ms.TotalIncome += o.Price * o.Amount
				ms.TotalSold += o.Amount
			}
		}
		ms.CarbonSaved += o.Amount * o.EnergySource.CarbonFactor()
		ms.TradeCount++
	}

	// Sort by month descending
	months := make([]string, 0, len(monthMap))
	for m := range monthMap {
		months = append(months, m)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(months)))

	result := make([]*TransactionSummary, 0, len(months))
	for _, m := range months {
		result = append(result, monthMap[m])
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}
