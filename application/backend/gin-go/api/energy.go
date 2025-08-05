package api

import (
	"encoding/json"
	"net/http"
	//"strconv"
	//"time"
	
	"gin-app/fabric"

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

func DeleteAsset(c *gin.Context) {
	id := c.Param("id")
	_, err := fabric.Contract.SubmitTransaction("DeleteAsset", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete asset" + id + ": " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, "Successfully delete.")
}

func RegisterUser(c *gin.Context) {
	// var userid, username, password string
	// var balance float64
	// var passwordhash string

	// userid = genuserid()

	// username = c.Param("username")
	// password = c.Param("password")
	// passwordhash = password
	// balance = 0.0
	// balanceStr := strconv.FormatFloat(balance, 'f', 2, 64) 

	// _, err := fabric.Contract.SubmitTransaction("RegisterUser", userid, username, passwordhash, balanceStr,)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register: " + err.Error()})
	// 	return
	// }

	_, err := fabric.Contract.SubmitTransaction("RegisterUser")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register: " + err.Error()})
		return
	}


	c.JSON(http.StatusOK, "Successfully Register.")

}