package api

import (
	//"fmt"
	"strconv"
	//"net/http"

	//"gin-app/fabric"

	//"github.com/gin-gonic/gin"
)

// func (s *SmartContract) RegisterUser(ctx contractapi.TransactionContextInterface, userid string, username string, passwordhash string, balance float64) error {
// 	user := User{
// 		UserID:       userid,
// 		UserName:     username,
// 		PasswordHash: passwordhash,
// 		Balance:      balance,	
// 	}

// 	userAsBytes, err := json.Marshal(user)
// 	if err != nil {
// 		return err
// 	}
// 	err = ctx.GetStub().PutState(userid, userAsBytes)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// var req Asset
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	_, err := fabric.Contract.SubmitTransaction("CreateAsset",
// 		req.ID, req.Color,
// 		strconv.Itoa(req.Size),
// 		req.Owner,
// 		strconv.Itoa(req.AppraisedValue),
// 	)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create asset: " + err.Error()})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{"message": "Asset created successfully"})




var CNT_USER int

func genuserid() (userid string){
	var number string

	CNT_USER += 1
	number = strconv.Itoa(CNT_USER)
	userid = "energy_user_"+number

	return userid
}

func genUseridInit() {
	CNT_USER = 0
}