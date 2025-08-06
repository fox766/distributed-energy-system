package mysql


type MysqlUser struct {
	UserName        string `json:"username"`
	UserID          string `json:"userid"`	
	PasswordHash    string `json:"passwordhash"`
}