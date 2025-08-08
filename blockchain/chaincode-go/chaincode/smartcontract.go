package chaincode

import (
	"encoding/json"
	"fmt"
	//"strconv"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)


type SmartContract struct {
	contractapi.Contract
}

// test


// Asset describes basic details of what makes up a simple asset
// Insert struct field in alphabetic order => to achieve determinism across languages
// golang keeps the order when marshal to json but doesn't order automatically
type Asset struct {
	AppraisedValue int    `json:"AppraisedValue"`
	Color          string `json:"Color"`
	ID             string `json:"ID"`
	Owner          string `json:"Owner"`
	Size           int    `json:"Size"`
}

// InitLedger adds a base set of assets to the ledger
func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	assets := []Asset{
		{ID: "asset1", Color: "blue", Size: 5, Owner: "Tomoko", AppraisedValue: 300},
		{ID: "asset2", Color: "red", Size: 5, Owner: "Brad", AppraisedValue: 400},
		{ID: "asset3", Color: "green", Size: 10, Owner: "Jin Soo", AppraisedValue: 500},
		{ID: "asset4", Color: "yellow", Size: 10, Owner: "Max", AppraisedValue: 600},
		{ID: "asset5", Color: "black", Size: 15, Owner: "Adriana", AppraisedValue: 700},
		{ID: "asset6", Color: "white", Size: 15, Owner: "Michel", AppraisedValue: 800},
	}
	
	for _, asset := range(assets) {
		assetJSON, err := json.Marshal(asset)
		if err != nil {
			return err
		}

		err = ctx.GetStub().PutState(asset.ID, assetJSON)
		if err != nil {
			return fmt.Errorf("failed to put to world state. %v", err)
		}
	}

	return nil
}

// ReadAsset returns the asset stored in the world state with given id.
func (s *SmartContract) ReadAsset(ctx contractapi.TransactionContextInterface, id string) (*Asset, error) {
	assetJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if assetJSON == nil {
		return nil, fmt.Errorf("the asset %s does not exist", id)
	}

	var asset Asset
	err = json.Unmarshal(assetJSON, &asset)
	if err != nil {
		return nil, err
	}

	return &asset, nil
}

// ========== 核心功能 ==========

// Init the energy price
func (s *SmartContract) Init(ctx contractapi.TransactionContextInterface) error{
	status := EnergyStatus{
		EnergyPrice:  1.0,
		Fee:          0.01,                 
	}

	statusJSON, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal: %v", err)
	}

	if err := ctx.GetStub().PutState("ENERGY_STATUS", paramsJSON); err != nil {
		return fmt.Errorf("failed to save to the world state %v", err)
	}

	return nil
}

// update the energy status
func (s *SmartContract) UpdateEnergyStatus(ctx contractapi.TransactionContextInterface, newstatus *EnergyStatus) error{
	newStatusJSON, err := json.Marshal(newStatus)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState("ENERGY_STATUS", assetJSON)
}

// register user
func (s *SmartContract) RegisterUser(ctx contractapi.TransactionContextInterface, userid, username, userrole string, 
	available, float64, balance float64) error {
	user := User{
		UserID:       userid,
		UserName:     username,
		UserRole:     userrole,
		Available:    available,
		Balance:      balance,	
	}

	userAsBytes, err := json.Marshal(user)
	if err != nil {
		return err
	}
	err = ctx.GetStub().PutState(userid, userAsBytes)
	if err != nil {
		return err
	}
	return nil
}

func (s *SmartContract) RegisterDevice(ctx contractapi.TransactionContextInterface) error {

}


func (s *SmartContract) TransferDevice(ctx contractapi.TransactionContextInterface, devicename, newowner string) error {

}

func (s *SmartContract) CreateOrder(ctx contractapi.TransactionContextInterface, id, partyA, partyB, status string, 
	createdtime time.time, amout, price float64) error {
	
	partyB = ""
	status = "CREATED"
	order = Order {
		ID: id,
		PartyA: partyA,
		partyB: partyB,
		Amount: amount,
		Price: price,
		Status: status,
		CreatedAt: createdtime,
	}

	orderJSON, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal: %v", err)
	}

	if err := ctx.GetStub().PutState(id, paramsJSON); err != nil {
		return fmt.Errorf("failed to save to the world state %v", err)
	}

	return nil
}

func (s *SmartContract) GetOrder(ctx contractapi.TransactionContextInterface, id string) (Order, error) {
	
	exists, err := s.ItemExists(ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("the order %s does not exist", id)
	}
	
	orderJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}

	var order Order
	err = json.Unmarshal(orderJSON, &order)
	if err != nil {
		return nil, err
	}

	return &order, nil
}

// change the status of order to MATCHED
func (s *SmartContract) MatchOrder(ctx contractapi.TransactionContextInterface, id, partyB string) error{
	order, err := s.GetOrder(ctx, id)
	if err != nil {
		return fmt.Errorf("get order %s failed", id)
	}
	order.PartyB = partyB
	if order.status != "CREATED" {
		return fmt.Errorf("Match order failed: %v", err)
	} 
	order.status = "MATCHED"
	orderJSON, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal: %v", err)
	}

	if err := ctx.GetStub().PutState(id, paramsJSON); err != nil {
		return fmt.Errorf("failed to save to the world state %v", err)
	}

	return nil
}

// settle the order
func (s *SmartContract) SettleOrder (ctx contractapi.TransactionContextInterface, )

// used to judge if the order, user, energy device exists or not
func (s *SmartContract) ItemExists(ctx contractapi.TransactionContextInterface, item string) (bool, error){
	itemJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}
	return itemJSON != nil, nil
}