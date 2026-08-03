package chaincode

import (
	"time"
)

// EnergySource represents the type of energy being traded
type EnergySource string

const (
	Solar   EnergySource = "SOLAR"
	Wind    EnergySource = "WIND"
	Storage EnergySource = "STORAGE"
)

// CarbonFactor returns the CO₂ emissions avoided per kWh for each energy source (kg/kWh).
func (e EnergySource) CarbonFactor() float64 {
	switch e {
	case Solar:
		return 0.6 // vs coal baseline
	case Wind:
		return 0.7
	case Storage:
		return 0.5
	default:
		return 0.3
	}
}

// User represents a participant in the energy trading system
type User struct {
	UserID    string  `json:"userid"`
	UserName  string  `json:"username"`
	UserRole  string  `json:"userrole"` // "PRODUCER" | "CONSUMER" | "admin"
	Available float64 `json:"available"`
	Balance   float64 `json:"balance"`
}

// EnergyStatus holds the global energy market parameters
type EnergyStatus struct {
	EnergyPrice float64 `json:"energyprice"`
	Fee         float64 `json:"fee"`
}

// Order represents an energy trade order on the ledger
type Order struct {
	ID            string       `json:"id"`
	PartyA        string       `json:"partyA"`
	PartyB        string       `json:"partyB"`
	Direction     string       `json:"direction"`    // "SELL" | "BUY"
	EnergySource  EnergySource `json:"energySource"`
	Amount        float64      `json:"amount"`       // kWh
	Price         float64      `json:"price"`
	Fee           float64      `json:"fee"`
	Status        string       `json:"status"`       // CREATED → MATCHED → FINISHED / CANCELLED
	DeliveryStart string       `json:"deliveryStart"`
	DeliveryEnd   string       `json:"deliveryEnd"`
	CreatedAt     time.Time    `json:"createdAt"`
}

// PriceRecord stores historical energy price data for charting
type PriceRecord struct {
	Timestamp time.Time `json:"timestamp"`
	Price     float64   `json:"price"`
	Volume    float64   `json:"volume"`
}

// CarbonRecord stores carbon savings per settlement
type CarbonRecord struct {
	OrderID     string    `json:"orderid"`
	Amount      float64   `json:"amount"`      // kWh
	Source      string    `json:"source"`      // SOLAR/WIND/STORAGE
	Coefficient float64   `json:"coefficient"` // kg/kWh
	CarbonSaved float64   `json:"carbonSaved"` // kg CO₂
	Timestamp   time.Time `json:"timestamp"`
}

// DeviceType represents the type of energy generation device
type DeviceType string

const (
	SolarPanel DeviceType = "SOLAR_PANEL"
	WindTurbine DeviceType = "WIND_TURBINE"
	BatteryStorage DeviceType = "BATTERY_STORAGE"
)

// GenerationRecord stores an energy generation event
type GenerationRecord struct {
	UserID    string    `json:"userid"`
	DeviceType DeviceType `json:"devicetype"`
	Amount    float64   `json:"amount"`    // kWh generated
	Timestamp time.Time `json:"timestamp"`
}

// TimeOfUsePeriod defines a pricing period
type TimeOfUsePeriod struct {
	Name       string  `json:"name"`       // "peak", "flat", "valley"
	StartHour  int     `json:"startHour"`  // 0-23
	EndHour    int     `json:"endHour"`    // 0-23
	Price      float64 `json:"price"`      // ¥/kWh
}

// TimeOfUseSchedule holds the TOU pricing configuration
type TimeOfUseSchedule struct {
	Periods   []TimeOfUsePeriod `json:"periods"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

// TransactionSummary holds monthly statement data for a user
type TransactionSummary struct {
	UserID        string  `json:"userid"`
	Month         string  `json:"month"`         // "2006-01"
	TotalIncome   float64 `json:"totalIncome"`   // from selling
	TotalExpense  float64 `json:"totalExpense"`  // from buying
	TotalSold     float64 `json:"totalSold"`     // kWh
	TotalBought   float64 `json:"totalBought"`   // kWh
	CarbonSaved   float64 `json:"carbonSaved"`   // kg CO₂
	TradeCount    int     `json:"tradeCount"`
}

// AuditEntry records every significant operation immutably on the ledger
type AuditEntry struct {
	TxID      string    `json:"txid"`
	Operation string    `json:"operation"`  // CreateOrder, MatchOrder, SettleOrder, CancelOrder, RegisterUser
	OrderID   string    `json:"orderid"`
	UserID    string    `json:"userid"`
	Details   string    `json:"details"`
	Timestamp time.Time `json:"timestamp"`
}

// Composite key prefixes for efficient ledger queries
const (
	OrderStatusPrefix     = "order~status"
	OrderUserPrefix       = "order~user"
	PriceHistoryPrefix    = "price~ts"
	AuditPrefix           = "audit~ts"
	CarbonPrefix          = "carbon~ts"
	GenerationPrefix      = "generation~ts"
	TransactionPrefix     = "txn~user"
	UserCountKey          = "USER_COUNT"
	OrderCountKey         = "ORDER_COUNT"
	EnergyStatusKey       = "ENERGY_STATUS"
	TOUScheduleKey        = "TOU_SCHEDULE"
	PriceHistoryCountKey  = "PRICE_HISTORY_COUNT"
	CarbonCountKey        = "CARBON_COUNT"
	GenerationCountKey    = "GENERATION_COUNT"
)
