package chaincode

import (
	"time"
)

type User struct {
	UserID       string          `json:"userid"`
	UserName     string          `json:"username"`
	UserRole     string          `json:"userrole"`
	Available    float64         `json:"available`
	Balance      float64         `json:"balance"`
	//DeviceList   []*EnergyDevice `json:"devicelist"`
}

type EnergyStatus struct {
	EnergyPrice  float64 `json:"energyprice"` 
	Fee          float64 `json:"fee"` 
}

type EnergyDevice struct {
	OwnerName       string  `json:"onwer`
	OwnerID         string  `json:"ownerid"`
	DeeviceName     string  `json:"devicename"`
}

type Order struct {
	ID             string    `json:"id"`
	PartyA         string    `json:"partyA"`    // seller
	PartyB         string    `json:"partyB"`    // buyer
 	Amount         float64   `json:"amount"`
	Price          float64   `json:"price"`
	Status         string    `json:"status"` // CREATED/MATCHED/COMPLETED
	CreatedAt      time.time `json:"createdAt"`
}