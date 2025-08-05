package chaincode

type User struct {
	UserID       string   `json:"userID"`
	UserName     string   `json:"username"`
	PasswordHash string   `json:"passwordHash"`
	Balance      float64  `json:"balance"`
}