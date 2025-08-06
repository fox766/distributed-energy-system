package api

import (
	"encoding/json"
	"net/http"
	//"time"
	
	"backend/gin-go/fabric"

	"github.com/gin-gonic/gin"
)





func InitLedger(c *gin.Context) {
	_, err := fabric.Contract.SubmitTransaction("InitLedger")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to Initialize asset: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, "Successfully Initialization.")
}



func ReadAsset(c *gin.Context) {
	id := c.Param("id")

	result, err := fabric.Contract.EvaluateTransaction("ReadAsset", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read asset: " + err.Error()})
		return
	}

	var asset Asset
	json.Unmarshal(result, &asset)

	c.JSON(http.StatusOK, asset)
}
// // 注册能源资产
// func (s *SmartContract) RegisterAsset(ctx contractapi.TransactionContextInterface, assetID string, capacity float64) error {
// 	clientID, err := s.GetClientIdentity(ctx)
// 	if err != nil {
// 		return fmt.Errorf("获取用户身份失败: %v", err)
// 	}

// 	asset := EnergyAsset{
// 		ID:        assetID,
// 		Owner:     clientID,
// 		Capacity:  capacity,
// 		Available: capacity,
// 	}

// 	assetJSON, err := json.Marshal(asset)
// 	if err != nil {
// 		return err
// 	}

// 	return ctx.GetStub().PutState(assetID, assetJSON)
// }

