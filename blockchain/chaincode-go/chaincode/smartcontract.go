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

// Init the energy price
func (s *SmartContract) Init(ctx contractapi.TransactionContextInterface) error{
	status := EnergyStatus{
		EnergyPrice:  1.0,
		Fee:          0.01,                 
	}

	statusJSON, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal: %v", err)
	}

	if err := ctx.GetStub().PutState("ENERGY_STATUS", statusJSON); err != nil {
		return fmt.Errorf("failed to save to the world state %v", err)
	}

	return nil
}

// get user info
func (s *SmartContract) GetUser(ctx contractapi.TransactionContextInterface, id string) (*User, error){
	exists, err := s.ItemExists(ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("the user %s does not exist", id)
	}
	
	userJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}

	var user User
	err = json.Unmarshal(userJSON, &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *SmartContract) GetEnergyStatus(ctx contractapi.TransactionContextInterface) (*EnergyStatus, error){
	energyJSON, err := ctx.GetStub().GetState("ENERGY_STATUS")
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}

	var energy EnergyStatus
	err = json.Unmarshal(energyJSON, &energy)
	if err != nil {
		return nil, err
	}
	return &energy, nil
}

func (s *SmartContract) GetOrder(ctx contractapi.TransactionContextInterface, id string) (*Order, error) {
	
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

func (s *SmartContract) UpdateUser(ctx contractapi.TransactionContextInterface, id string, available, balance float64) error{
	user, err := s.GetUser(ctx, id)
	if (err != nil) {
		return fmt.Errorf("failed to get user while updating user info: %v", err)
	}

	// if available or balance < 0, do not update
	if available >= 0 {
		user.Available = available
	}
	if balance >= 0 {
		user.Balance = balance
	}

	userJSON, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal: %v", err)
	}

	if err := ctx.GetStub().PutState(id, userJSON); err != nil {
		return fmt.Errorf("failed to save to the world state %v", err)
	}
	return nil
}

func (s *SmartContract) UpdateOrder(ctx contractapi.TransactionContextInterface, id, status string) error{
	order, err := s.GetOrder(ctx, id)
	if (err != nil) {
		return fmt.Errorf("failed to get order while updating order info: %v", err)
	}
	order.Status = status

	orderJSON, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal: %v", err)
	}

	if err := ctx.GetStub().PutState(id, orderJSON); err != nil {
		return fmt.Errorf("failed to save to the world state %v", err)
	}
	return nil
}

// update the energy status: Price and Fee
func (s *SmartContract) UpdateEnergyStatus(ctx contractapi.TransactionContextInterface, price, fee float64) error{
	newstatus := EnergyStatus{
		EnergyPrice:  price,
		Fee:          fee,                 
	}
	newstatusJSON, err := json.Marshal(newstatus)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState("ENERGY_STATUS", newstatusJSON)
}

// register user
func (s *SmartContract) RegisterUser(ctx contractapi.TransactionContextInterface, userid, username, userrole string, 
	available, balance float64) error {
	user := User{
		UserID:       userid,
		UserName:     username,
		UserRole:     userrole,
		Available:    available,
		Balance:      balance,	
	}

	userJSON, err := json.Marshal(user)
	if err != nil {
		return err
	}
	err = ctx.GetStub().PutState(userid, userJSON)
	if err != nil {
		return err
	}
	return nil
}

func (s *SmartContract) RegisterDevice(ctx contractapi.TransactionContextInterface) error {
	return nil
}


func (s *SmartContract) TransferDevice(ctx contractapi.TransactionContextInterface, devicename, newowner string) error {
	return nil
}

func (s *SmartContract) CreateOrder(ctx contractapi.TransactionContextInterface, id, partyA, partyB, status string, 
	createdtime time.Time, amount, price, fee float64) error {
	
	userA, err := s.GetUser(ctx, partyA)
	if err != nil {
		return fmt.Errorf("failed to find partyA while creating order: %v", err)
	}
	if userA.Available < amount {
		return fmt.Errorf("failed to create order, for partyA's available is insufficient: %v", err)
	}
	partyB = ""
	status = "CREATED"
	order := Order {
		ID: id,
		PartyA: partyA,
		PartyB: partyB,
		Amount: amount,
		Price: price,
		Fee: fee,
		Status: status,
		CreatedAt: createdtime,
	}

	orderJSON, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal: %v", err)
	}

	if err := ctx.GetStub().PutState(id, orderJSON); err != nil {
		return fmt.Errorf("failed to save to the world state %v", err)
	}

	userA.Available -= amount
	err = s.UpdateUser(ctx, userA.UserID, userA.Available, userA.Balance)
	if err != nil {
		return fmt.Errorf("failed to create order, for failing to update partyA's info: %v", err)
	}

	return nil
}


// change the status of order to MATCHED
func (s *SmartContract) MatchOrder(ctx contractapi.TransactionContextInterface, id, partyB string) error{
	order, err := s.GetOrder(ctx, id)
	if err != nil {
		return fmt.Errorf("get order %s failed", id)
	}
	order.PartyB = partyB
	if order.Status != "CREATED" {
		return fmt.Errorf("Match order failed: %v", err)
	} 
	order.Status = "MATCHED"
	orderJSON, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal: %v", err)
	}

	if err := ctx.GetStub().PutState(id, orderJSON); err != nil {
		return fmt.Errorf("failed to save to the world state %v", err)
	}

	return nil
}


// settle the order
func (s *SmartContract) SettleOrder(ctx contractapi.TransactionContextInterface, id string) error{
	var order *Order
	var err error
	var userA, userB *User

	order, err = s.GetOrder(ctx, id) 
	if err != nil {
		return err
	}
	if order.Status != "MATCHED" {
		return fmt.Errorf("Incorrect order status.")
	}

	userAid := order.PartyA
	userBid := order.PartyB
	userA, err = s.GetUser(ctx, userAid)
	if err != nil {
		return err
	}
	userB, err = s.GetUser(ctx, userBid)
	if err != nil {
		return err
	}
	
	altogether := order.Price * order.Amount
	if userB.Balance < altogether {
		// change the order to "Created" status
		status := "CREATED"
		err := s.UpdateOrder(ctx, order.ID, status)
		if err != nil {
			return err
		}
		return fmt.Errorf("trade failed, your balance is not enough.")
	}

	userB.Balance -= altogether
	userA.Balance += altogether
	if err = s.UpdateUser(ctx, userA.UserID, userA.Available, userA.Balance); err != nil {
		return err
	}
	if err = s.UpdateUser(ctx, userB.UserID, userB.Available, userB.Balance); err != nil {
		return err
	}

	order.Status = "FINISHED"
	orderJSON, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal: %v", err)
	}

	if err := ctx.GetStub().PutState(id, orderJSON); err != nil {
		return fmt.Errorf("failed to save to the world state %v", err)
	}

	//
	return nil
}

// change the order to "CANCELLED" STATUS and return the AMOUNT to PartyA's Available
func (s *SmartContract) CancelOrder(ctx contractapi.TransactionContextInterface, id string) error {
	var order *Order
	var err error
	var userA *User

	order, err = s.GetOrder(ctx, id) 
	if err != nil {
		return err
	}
	order.Status = "CANCELLED"
	
	userAid := order.PartyA
	userA, err = s.GetUser(ctx, userAid)
	if err != nil {
		return err
	}
	userA.Available += order.Amount
	err = s.UpdateUser(ctx, userAid, userA.Available, userA.Balance) 
	if err != nil {
		return err
	}
	err = s.UpdateOrder(ctx, order.ID, order.Status)
	if err != nil {
		return err
	}
	return nil
}

// used to judge if the order, user, energy device exists or not
func (s *SmartContract) ItemExists(ctx contractapi.TransactionContextInterface, item string) (bool, error){
	itemJSON, err := ctx.GetStub().GetState(item)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}
	return itemJSON != nil, nil
}

