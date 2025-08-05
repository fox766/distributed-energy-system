package api

type Asset struct {
	ID             string `json:"ID"`
	Color          string `json:"Color"`
	Size           int    `json:"Size"`
	Owner          string `json:"Owner"`
	AppraisedValue int    `json:"AppraisedValue"`
}


type User struct {
	UserID       string   `json:"userID"`
	UserName     string   `json:"username"`
	PasswordHash string   `json:"passwordHash"`
	Balance      float64  `json:"balance"`
}

type registerUser struct {
	username string `json:"username"`
	password string `json:password"`
}