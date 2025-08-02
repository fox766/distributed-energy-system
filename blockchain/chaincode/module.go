package chaincode

// 能源资产结构
type EnergyAsset struct {
	ID        string  `json:"id"`
	Owner     string  `json:"owner"`
	Capacity  float64 `json:"capacity"` // 总产能(kWh)
	Available float64 `json:"available"` // 可交易量
}

// 交易订单结构
type TradeOrder struct {
	ID         string    `json:"id"`
	Seller     string    `json:"seller"`
	Buyer      string    `json:"buyer"`
	Amount     float64   `json:"amount"`
	Price      float64   `json:"price"`     // 积分/kWh
	Status     string    `json:"status"`    // CREATED/CONFIRMED/COMPLETED
	CreatedAt  time.Time `json:"createdAt"` // 创建时间
	ConfirmedAt time.Time `json:"confirmedAt"` // 确认时间
}

// 用户账户
type UserAccount struct {
	UserID  string  `json:"userId"`
	Balance float64 `json:"balance"` // 能源积分
}
