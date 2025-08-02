package chaincode

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)


type SmartContract struct {
	contractapi.Contract
}

// ========== 核心功能 ==========

// 注册能源资产
func (s *SmartContract) RegisterAsset(ctx contractapi.TransactionContextInterface, assetID string, capacity float64) error {
	clientID, err := s.GetClientIdentity(ctx)
	if err != nil {
		return fmt.Errorf("获取用户身份失败: %v", err)
	}

	asset := EnergyAsset{
		ID:        assetID,
		Owner:     clientID,
		Capacity:  capacity,
		Available: capacity,
	}

	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(assetID, assetJSON)
}

// 创建交易订单
func (s *SmartContract) CreateOrder(ctx contractapi.TransactionContextInterface, 
	orderID string, seller string, amount float64, price float64) error {
	
	clientID, err := s.GetClientIdentity(ctx)
	if err != nil {
		return err
	}

	// 验证卖方身份
	if clientID != seller {
		return fmt.Errorf("无权为他人创建订单")
	}

	// 检查卖方能源可用性
	sellerAsset, err := s.GetAsset(ctx, seller+"_asset")
	if err != nil || sellerAsset.Available < amount {
		return fmt.Errorf("能源可用量不足")
	}

	order := TradeOrder{
		ID:        orderID,
		Seller:    seller,
		Amount:    amount,
		Price:     price,
		Status:    "CREATED",
		CreatedAt: time.Now(),
	}

	orderJSON, err := json.Marshal(order)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(orderID, orderJSON)
}

// 确认交易
func (s *SmartContract) ConfirmTrade(ctx contractapi.TransactionContextInterface, 
	orderID string, buyer string) error {
	
	// 获取订单
	orderJSON, err := ctx.GetStub().GetState(orderID)
	if err != nil {
		return fmt.Errorf("读取订单失败: %v", err)
	}
	if orderJSON == nil {
		return fmt.Errorf("订单不存在")
	}

	var order TradeOrder
	json.Unmarshal(orderJSON, &order)

	// 验证买方身份
	clientID, err := s.GetClientIdentity(ctx)
	if err != nil {
		return err
	}
	if clientID != buyer {
		return fmt.Errorf("无权确认他人订单")
	}

	// 更新订单状态
	order.Buyer = buyer
	order.Status = "CONFIRMED"
	order.ConfirmedAt = time.Now()

	// 执行能源转移
	if err := s.transferEnergy(ctx, order.Seller, order.Buyer, order.Amount); err != nil {
		return err
	}

	// 执行积分结算
	if err := s.settlePayment(ctx, order.Buyer, order.Seller, order.Amount*order.Price); err != nil {
		return err
	}

	// 更新订单状态
	order.Status = "COMPLETED"
	updatedOrder, _ := json.Marshal(order)
	
	// 保存状态并发送事件
	ctx.GetStub().PutState(orderID, updatedOrder)
	ctx.GetStub().SetEvent("TradeCompleted", []byte(orderID))
	
	return nil
}

// ========== 辅助函数 ==========
func (s *SmartContract) transferEnergy(ctx contractapi.TransactionContextInterface, 
	seller string, buyer string, amount float64) error {
	
	// 更新卖方资产
	sellerAsset, err := s.GetAsset(ctx, seller+"_asset")
	if err != nil {
		return err
	}
	sellerAsset.Available -= amount
	sellerAssetJSON, _ := json.Marshal(sellerAsset)
	ctx.GetStub().PutState(seller+"_asset", sellerAssetJSON)

	// 更新买方资产（如果不存在则创建）
	buyerAsset, err := s.GetAsset(ctx, buyer+"_asset")
	if err != nil {
		buyerAsset = &EnergyAsset{
			ID:        buyer + "_asset",
			Owner:     buyer,
			Capacity:  0,
			Available: amount,
		}
	} else {
		buyerAsset.Available += amount
	}
	buyerAssetJSON, _ := json.Marshal(buyerAsset)
	ctx.GetStub().PutState(buyer+"_asset", buyerAssetJSON)
	
	return nil
}

func (s *SmartContract) GetClientIdentity(ctx contractapi.TransactionContextInterface) (string, error) {
	id, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return "", err
	}
	// 简化处理：实际应用中应提取证书中的CN
	return id, nil
}