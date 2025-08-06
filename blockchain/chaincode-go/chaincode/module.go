package chaincode

type User struct {
	UserID       string   `json:"userID"`
	UserName     string   `json:"username"`
	Balance      float64  `json:"balance"`
}