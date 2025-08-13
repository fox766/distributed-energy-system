package mysql


type MysqlUser struct {
	UserName        string `json:"username"`
	UserID          string `json:"userid"`	
	PasswordHash    string `json:"passwordhash"`
}

type MysqlOrder struct {
	OrderID string    `json:"orderid"`
    PartyA  string	  `json:"partyA"`
    PartyB  string    `json:"partyB"`
    Status  string    `json:"status"`
    Amount  float64   `json:"amount"`
}