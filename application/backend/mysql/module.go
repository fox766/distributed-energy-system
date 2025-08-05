package mysql

// user_id INT PRIMARY KEY ,
// username VARCHAR(50) UNIQUE NOT NULL,
// `password` VARCHAR(50) NOT NULL,
// RealInfo VARCHAR(100)

type MysqlUser struct {
	UserID          string `json:"userid"`
	UserName        string `json:"username"`
	PasswordHash    string `json:"passwordhash"`
}