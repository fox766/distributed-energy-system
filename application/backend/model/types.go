package model

import "time"

// --- Request DTOs ---

type RegisterRequest struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
	Role        string `json:"role" binding:"required"`
	EnergyType  string `json:"energyType" binding:"required"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type CreateOrderRequest struct {
	Amount       float64 `json:"amount" binding:"required"`
	Price        float64 `json:"price" binding:"required"`
	EnergySource string  `json:"energySource" binding:"required"`
	DeliveryStart string `json:"deliveryStart" binding:"required"`
	DeliveryEnd   string  `json:"deliveryEnd" binding:"required"`
	Direction    string  `json:"direction"` // "SELL" | "BUY", default "SELL"
}

// UpdateUserRequest uses pointers so that an omitted field ("leave as is") is
// distinguishable from an explicit zero. With plain float64s an empty {} body
// was indistinguishable from {"available":0} and silently wiped the balance.
type UpdateUserRequest struct {
	Available *float64 `json:"available"`
	Balance   *float64 `json:"balance"`
}

// TopUpRequest credits a user's balance. Admin-only (see the /api/admin group).
type TopUpRequest struct {
	UserID string  `json:"userid" binding:"required"`
	Amount float64 `json:"amount" binding:"required"`
}

type UpdateEnergyRequest struct {
	Price float64 `json:"price" binding:"required"`
	Fee   float64 `json:"fee" binding:"required"`
}

// --- Response DTOs ---

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

// --- Business entities (mirror chaincode) ---

type User struct {
	UserID     string  `json:"userid"`
	UserName   string  `json:"username"`
	UserRole   string  `json:"userrole"`
	Available  float64 `json:"available"`
	Balance    float64 `json:"balance"`
	EnergyType string  `json:"energytype,omitempty"`
}

type Order struct {
	ID            string    `json:"id"`
	PartyA        string    `json:"partyA"`
	PartyB        string    `json:"partyB"`
	Direction     string    `json:"direction"`
	EnergySource  string    `json:"energySource"`
	Amount        float64   `json:"amount"`
	Price         float64   `json:"price"`
	Fee           float64   `json:"fee"`
	Status        string    `json:"status"`
	DeliveryStart string    `json:"deliveryStart"`
	DeliveryEnd   string    `json:"deliveryEnd"`
	CreatedAt     time.Time `json:"createdAt"`
	MatchedWith   string    `json:"matchedWith,omitempty"`
}

type EnergyStatus struct {
	EnergyPrice float64 `json:"energyprice"`
	Fee         float64 `json:"fee"`
}

type SystemStatus struct {
	EnergyPrice float64 `json:"energyprice"`
	Fee         float64 `json:"fee"`
	UserNum     int     `json:"usernum"`
	OrderNum    int     `json:"ordernum"`
}

type PriceRecord struct {
	Timestamp time.Time `json:"timestamp"`
	Price     float64   `json:"price"`
	Volume    float64   `json:"volume"`
}

// --- P0: TOU Pricing ---

type TOUPriceResponse struct {
	Price  float64 `json:"price"`
	Period string  `json:"period"`
	Hour   int     `json:"hour"`
}

// --- P0: Generation ---

type GenerateRequest struct {
	DeviceType string `json:"deviceType" binding:"required"` // SOLAR_PANEL, WIND_TURBINE, BATTERY_STORAGE
}

type GenerationRecord struct {
	UserID     string  `json:"userid"`
	DeviceType string  `json:"devicetype"`
	Amount     float64 `json:"amount"`
	Timestamp  string  `json:"timestamp"`
}

// --- P2: Auto-Match ---

type AutoMatchResponse struct {
	Matched   bool   `json:"matched"`
	Message   string `json:"message,omitempty"`
	SellOrder string `json:"sellOrder,omitempty"`
	BuyOrder  string `json:"buyOrder,omitempty"`
	Price     float64 `json:"price,omitempty"`
	Amount    float64 `json:"amount,omitempty"`
}

// BatchMatchResponse reports a drain of the order book. When it comes straight
// from the chaincode, Matched is 0 or 1 (one pair per transaction); the backend
// loop sums those into the total it returns to the client.
type BatchMatchResponse struct {
	Matched   int `json:"matched"`
	Remaining int `json:"remaining"`
}

// MatchOrderRequest names the counterpart ORDER (not user) to pair with.
type MatchOrderRequest struct {
	CounterpartyOrderID string `json:"counterpartyOrderId" binding:"required"`
}

// --- P3: Transaction History ---

type TransactionHistoryRequest struct {
	UserID string `json:"userid"`
	Month  string `json:"month"`
	Limit  int    `json:"limit"`
}

type MonthlySummary struct {
	UserID       string  `json:"userid"`
	Month        string  `json:"month"`
	TotalIncome  float64 `json:"totalIncome"`
	TotalExpense float64 `json:"totalExpense"`
	TotalSold    float64 `json:"totalSold"`
	TotalBought  float64 `json:"totalBought"`
	CarbonSaved  float64 `json:"carbonSaved"`
	TradeCount   int     `json:"tradeCount"`
}

// --- JWT Claims ---

type Claims struct {
	UserID   string `json:"uid"`
	UserRole string `json:"role"`
}
