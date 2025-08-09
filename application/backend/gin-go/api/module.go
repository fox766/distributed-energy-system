package api

import (
	"time"
)

type Asset struct {
	ID             string `json:"ID"`
	Color          string `json:"Color"`
	Size           int    `json:"Size"`
	Owner          string `json:"Owner"`
	AppraisedValue int    `json:"AppraisedValue"`
}


type User struct {
	UserID       string          `json:"userid"`
	UserName     string          `json:"username"`
	UserRole     string          `json:"userrole"`
	Available    float64         `json:"available`
	Balance      float64         `json:"balance"`
	//DeviceList   []*EnergyDevice `json:"devicelist"`
}

type Order struct {
	ID             string    `json:"id"`
	PartyA         string    `json:"partyA"`    // seller
	PartyB         string    `json:"partyB"`    // buyer
 	Amount         float64   `json:"amount"`
	Price          float64   `json:"price"`
	Fee            float64   `json:"fee"`
	Status         string    `json:"status"` // CREATED/MATCHED/COMPLETED
	CreatedAt      time.Time `json:"createdAt"`
}

type EnergyStatus struct {
	EnergyPrice  float64 `json:"energyprice"` 
	Fee          float64 `json:"fee"` 
}