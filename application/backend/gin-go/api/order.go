package api 

import (
	"strconv"
	"time"
	"net/http"
	"encoding/json"

	"backend/jwt"
	"backend/gin-go/fabric"

	"github.com/gin-gonic/gin"
)

var CNT_ORDER int

func genorderid() (orderid string){
	var number string

	CNT_ORDER += 1
	number = strconv.Itoa(CNT_ORDER)
	orderid = "energy_order_"+number

	return orderid
}

func OrderInit() {
	CNT_ORDER = 0
}

//func (s *SmartContract) CreateOrder(ctx contractapi.TransactionContextInterface, id, partyA, partyB, status string, 
//	createdtime time.Time, amount, price, fee float64) error {
func CreateOrder(c *gin.Context) {
	if !UserLogged(){
		c.JSON(http.StatusOK, gin.H{"message": "NO user logged in, cannot create a order",})
	}
	
	var userinfo *jwt.LoginUser
	var err error
	userinfo, err = jwt.ParseToken(CURRENT_USER)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "Failed to parse token",})
	}

	orderid := genorderid()
	partyA := userinfo.UserID
	partyB := ""
	status := "CREATED"
	createdtime := time.Now()
	amount := c.Param("amount")
	result, err := fabric.Contract.EvaluateTransaction("GetEnergyStatus")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read energystatus: " + err.Error()})
		return
	}

	var energy EnergyStatus
	json.Unmarshal(result, &energy)

	price := energy.EnergyPrice
	fee := energy.Fee
	priceStr := strconv.FormatFloat(price, 'f', 2, 64)
	feeStr := strconv.FormatFloat(fee, 'f', 2, 64) 
	timeStr := createdtime.Format(time.RFC3339) 
	_, err = fabric.Contract.SubmitTransaction("CreateOrder", orderid, partyA, partyB, status, timeStr, amount, priceStr, feeStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, "Successfully create a order." + "Your orderid is " + orderid)

}