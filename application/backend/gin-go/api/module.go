package api

import (
)

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
	Balance      float64  `json:"balance"`
}

