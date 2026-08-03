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
func (s *SmartContract) MatchOrder(ctx contractapi.TransactionContextInterface,
	id, partyB string) error {

	order, err := s.GetOrder(ctx, id)
	if err != nil {
		return fmt.Errorf("get order %s failed: %v", id, err)
	}
	if order.Status != "CREATED" {
		return fmt.Errorf("order %s has status %s, expected CREATED", id, order.Status)
	}
	if partyB == order.PartyA {
		return fmt.Errorf("cannot match your own order")
	}

	// For BUY orders, buyer must have enough balance
	if order.Direction == "BUY" {
		buyer, err := s.GetUser(ctx, partyB)
		if err != nil {
			return err
		}
		totalCost := order.Amount * order.Price
		if buyer.Balance < totalCost {
			return fmt.Errorf("buyer has insufficient balance: have ¥%.2f, need ¥%.2f",
				buyer.Balance, totalCost)
		}
	}

	// Remove old composite key
	oldStatusKey, _ := ctx.GetStub().CreateCompositeKey(OrderStatusPrefix, []string{order.Status, id})
	ctx.GetStub().DelState(oldStatusKey)

	order.PartyB = partyB
	order.Status = "MATCHED"

	orderJSON, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal order: %v", err)
	}
	if err := ctx.GetStub().PutState(id, orderJSON); err != nil {
		return fmt.Errorf("failed to save order: %v", err)
	}

	// New composite key for MATCHED status
	newStatusKey, _ := ctx.GetStub().CreateCompositeKey(OrderStatusPrefix, []string{order.Status, id})
	ctx.GetStub().PutState(newStatusKey, []byte{0})

	// Composite key for buyer
	userKey, _ := ctx.GetStub().CreateCompositeKey(OrderUserPrefix, []string{partyB, id})
	ctx.GetStub().PutState(userKey, []byte{0})

	s.recordAudit(ctx, "MatchOrder", id, partyB,
		fmt.Sprintf("matched with seller=%s", order.PartyA))

	ctx.GetStub().SetEvent("OrderMatched", orderJSON)
	return nil
}

// SettleOrder completes a MATCHED order, transferring funds and energy.
func (s *SmartContract) SettleOrder(ctx contractapi.TransactionContextInterface, id string) error {
	order, err := s.GetOrder(ctx, id)
	if err != nil {
		return err
	}
	if order.Status != "MATCHED" {
		return fmt.Errorf("order %s has status %s, expected MATCHED", id, order.Status)
	}

	userA, err := s.GetUser(ctx, order.PartyA)
	if err != nil {
		return err
	}
	userB, err := s.GetUser(ctx, order.PartyB)
	if err != nil {
		return err
	}

	totalAmount := order.Price * order.Amount

	if order.Direction == "SELL" {
		// Seller (partyA) gets money, Buyer (partyB) gets energy
		if userB.Balance < totalAmount {
			// Revert order to CREATED
			s.revertOrderToCreated(ctx, order)
			return fmt.Errorf("trade failed: buyer balance ¥%.2f insufficient for ¥%.2f",
				userB.Balance, totalAmount)
		}
		userB.Balance -= totalAmount
		userA.Balance += totalAmount
		userB.Available += order.Amount
	} else {
		// BUY order: partyA already paid at creation; partyB (seller) gets money, partyA gets energy
		userB.Balance += totalAmount
		userA.Available += order.Amount
	}

	if err := s.saveUser(ctx, userA); err != nil {
		return err
	}
	if err := s.saveUser(ctx, userB); err != nil {
		return err
	}

	// Update composite keys
	oldStatusKey, _ := ctx.GetStub().CreateCompositeKey(OrderStatusPrefix, []string{order.Status, id})
	ctx.GetStub().DelState(oldStatusKey)

	order.Status = "FINISHED"
	orderJSON, _ := json.Marshal(order)
	ctx.GetStub().PutState(id, orderJSON)

	newStatusKey, _ := ctx.GetStub().CreateCompositeKey(OrderStatusPrefix, []string{order.Status, id})
	ctx.GetStub().PutState(newStatusKey, []byte{0})

	// Record price history
	s.recordPriceHistory(ctx, order.Price, order.Amount)

	// Record carbon savings
	s.recordCarbon(ctx, order)

	s.recordAudit(ctx, "SettleOrder", id, userB.UserID,
		fmt.Sprintf("settled %.0fkWh @ ¥%.2f, carbon saved: %.2f kg",
			order.Amount, order.Price, order.Amount*order.EnergySource.CarbonFactor()))

	ctx.GetStub().SetEvent("OrderSettled", orderJSON)
	return nil
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

// now returns the transaction timestamp deterministically (same for all endorsing peers).
func (s *SmartContract) now(ctx contractapi.TransactionContextInterface) time.Time {
	ts, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return time.Unix(0, 0)
	}
	t := time.Unix(ts.Seconds, int64(ts.Nanos))
	return t
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


// revertOrderToCreated reverts a MATCHED order back to CREATED status
// (used when settlement fails).
func (s *SmartContract) revertOrderToCreated(ctx contractapi.TransactionContextInterface,
	order *Order) {

	oldKey, _ := ctx.GetStub().CreateCompositeKey(OrderStatusPrefix, []string{order.Status, order.ID})
	ctx.GetStub().DelState(oldKey)

	order.Status = "CREATED"
	order.PartyB = ""
	orderJSON, _ := json.Marshal(order)
	ctx.GetStub().PutState(order.ID, orderJSON)

	newKey, _ := ctx.GetStub().CreateCompositeKey(OrderStatusPrefix, []string{order.Status, order.ID})
	ctx.GetStub().PutState(newKey, []byte{0})
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
	default:
		generated = 1.0
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

	oppositeDir := "BUY"
	if order.Direction == "BUY" {
		oppositeDir = "SELL"
	}

	type matchScore struct {
		candidate *Order
		score     float64
		overlap   bool
	}
	var best *matchScore

	for _, candidate := range allOrders {
		if candidate.ID == orderID {
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

		// Delivery window overlap check
		overlap := deliveryWindowsOverlap(order, candidate)
		// Price spread (smaller is better: buyer wants low sell price)
		score := buyPrice - sellPrice
		if overlap {
			score += 10.0 // heavy bonus for overlapping delivery windows
		}

		if best == nil || score > best.score ||
			(score == best.score && !best.overlap && overlap) {
			best = &matchScore{candidate: candidate, score: score, overlap: overlap}
		}
	}

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

	// Partial fill: trade the minimum of the two amounts
	tradeAmount := sellOrder.Amount
	if buyOrder.Amount < tradeAmount {
		tradeAmount = buyOrder.Amount
	}

	// If full amounts match, do a simple match & settle
	if sellOrder.Amount == buyOrder.Amount {
		return s.settleMatchPair(ctx, sellOrder, buyOrder, tradeAmount)
	}

	// Partial fill — need to split the larger order
	partial := false
	var residual *Order
	if sellOrder.Amount > buyOrder.Amount {
		// Split sell order: create residual SELL for remaining amount
		residualID := sellOrder.ID + "_r"
		residualAmount := sellOrder.Amount - buyOrder.Amount
		err = s.CreateOrder(ctx, residualID, sellOrder.PartyA, "SELL",
			string(sellOrder.EnergySource), sellOrder.DeliveryStart, sellOrder.DeliveryEnd,
			residualAmount, sellOrder.Price, sellOrder.Fee)
		if err != nil {
			return "", fmt.Errorf("partial fill: failed to create residual sell order: %v", err)
		}
		residual, _ = s.GetOrder(ctx, residualID)
		partial = true
	} else if buyOrder.Amount > sellOrder.Amount {
		// Split buy order: create residual BUY for remaining amount
		residualID := buyOrder.ID + "_r"
		residualAmount := buyOrder.Amount - sellOrder.Amount
		err = s.CreateOrder(ctx, residualID, buyOrder.PartyA, "BUY",
			string(buyOrder.EnergySource), buyOrder.DeliveryStart, buyOrder.DeliveryEnd,
			residualAmount, buyOrder.Price, buyOrder.Fee)
		if err != nil {
			return "", fmt.Errorf("partial fill: failed to create residual buy order: %v", err)
		}
		residual, _ = s.GetOrder(ctx, residualID)
		partial = true
	}

	result, err := s.settleMatchPair(ctx, sellOrder, buyOrder, tradeAmount)
	if err != nil {
		return "", err
	}

	if partial {
		result = result[:len(result)-1] + fmt.Sprintf(`,"partial":true,"residualOrder":"%s","tradedAmount":%.0f}`, residual.ID, tradeAmount)
	}

	s.recordAudit(ctx, "AutoMatch", orderID, "",
		fmt.Sprintf("auto-matched SELL:%s + BUY:%s amount=%.0fkWh @ ¥%.2f delivery_overlap=%v partial=%v",
			sellOrder.ID, buyOrder.ID, tradeAmount, sellOrder.Price, best.overlap, partial))

	return result, nil
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

// settleMatchPair matches and settles a pair of orders.
func (s *SmartContract) settleMatchPair(ctx contractapi.TransactionContextInterface,
	sellOrder, buyOrder *Order, amount float64) (string, error) {

	if err := s.MatchOrder(ctx, sellOrder.ID, buyOrder.PartyA); err != nil {
		return "", fmt.Errorf("auto-match sell order failed: %v", err)
	}
	if err := s.MatchOrder(ctx, buyOrder.ID, sellOrder.PartyA); err != nil {
		return "", fmt.Errorf("auto-match buy order failed: %v", err)
	}
	if err := s.SettleOrder(ctx, sellOrder.ID); err != nil {
		return "", fmt.Errorf("auto-settle sell order failed: %v", err)
	}
	if err := s.SettleOrder(ctx, buyOrder.ID); err != nil {
		return "", fmt.Errorf("auto-settle buy order failed: %v", err)
	}

	return fmt.Sprintf(`{"matched":true,"sellOrder":"%s","buyOrder":"%s","price":%.2f,"amount":%.0f}`,
		sellOrder.ID, buyOrder.ID, sellOrder.Price, amount), nil
}

// RunAutoMatch scans all CREATED orders and attempts to match them.
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

	matched := 0
	for _, sell := range sellOrders {
		if sell.Status != "CREATED" {
			continue
		}
		for _, buy := range buyOrders {
			if buy.Status != "CREATED" {
				continue
			}
			if sell.PartyA == buy.PartyA {
				continue
			}
			if sell.Price > buy.Price {
				continue
			}
			// Match and settle
			if err := s.MatchOrder(ctx, sell.ID, buy.PartyA); err != nil {
				continue
			}
			if err := s.MatchOrder(ctx, buy.ID, sell.PartyA); err != nil {
				continue
			}
			if err := s.SettleOrder(ctx, sell.ID); err != nil {
				continue
			}
			if err := s.SettleOrder(ctx, buy.ID); err != nil {
				continue
			}
			matched++
			s.recordAudit(ctx, "AutoMatch", sell.ID, "",
				fmt.Sprintf("batch-matched SELL %s + BUY %s @ ¥%.2f", sell.ID, buy.ID, sell.Price))
			break // move to next sell order
		}
	}

	return fmt.Sprintf(`{"matched":%d,"remaining":%d}`, matched, len(allOrders)-matched*2), nil
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
