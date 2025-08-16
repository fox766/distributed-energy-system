package api 

import (
	"strconv"
	"time"
	"net/http"
	"encoding/json"

	"backend/jwt"
	"backend/gin-go/fabric"
	"backend/mysql"

	"github.com/gin-gonic/gin"
)

var CNT_ORDER int
var ORDER_ID_USED bool

func genorderid() (orderid string){
	var number string

	if ORDER_ID_USED {
		CNT_ORDER += 1
	}
	number = strconv.Itoa(CNT_ORDER)
	orderid = "energy_order_"+number
	ORDER_ID_USED = true
	return orderid
}

func OrderInit() {
	CNT_ORDER = 0
	ORDER_ID_USED = true
}

func OrderNotUsed() {
	ORDER_ID_USED = false
}

func GetOrder(c *gin.Context) {
	var orderid string
	orderid = c.Param("orderid")
	result, err := fabric.Contract.EvaluateTransaction("GetOrder", orderid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read order: " + err.Error()})
		return
	}

	var order Order
	json.Unmarshal(result, &order)
	c.JSON(http.StatusOK, order)
	c.String(200, "\n")
}

func CreateOrder(c *gin.Context) {
	if !UserLogged(){
		c.JSON(http.StatusOK, gin.H{"message": "NO user logged in, cannot create a order",})
		return 
	}
	
	var userinfo *jwt.LoginUser
	var err error
	userinfo, err = jwt.ParseToken(CURRENT_USER)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "Failed to parse token" + CURRENT_USER,})
		return
	}


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
	orderid := genorderid()

	_, err = fabric.Contract.SubmitTransaction("CreateOrder", orderid, partyA, partyB, status, timeStr, amount, priceStr, feeStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order: " + err.Error()})
		OrderNotUsed()
		return
	}

	amountValue, _ := strconv.ParseFloat(amount, 64)
	err = mysql.InsertOrder(orderid, partyA, partyB, status, amountValue)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert order into database: " + err.Error()})
		OrderNotUsed()
		return
	}

	c.JSON(http.StatusOK, "Successfully create a order." + "Your orderid is " + orderid)
	c.String(200, "\n")

}

func MatchOrder(c *gin.Context) {
	orderid := c.Param("orderid")
	if !UserLogged(){
		c.JSON(http.StatusOK, gin.H{"message": "NO user logged in, cannot match the order",})
		return 
	}

	var userinfo *jwt.LoginUser
	var err error
	userinfo, err = jwt.ParseToken(CURRENT_USER)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "Failed to parse token" + CURRENT_USER,})
		return
	}
	partyB := userinfo.UserID

	_, err = fabric.Contract.SubmitTransaction("MatchOrder", orderid, partyB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to match order: " + err.Error()})
		return
	}

	err = mysql.UpdateOrderStatus(orderid, "MATCHED")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "Failed to change order status",})
		return 
	}

	err = mysql.UpdateOrderParty(orderid, partyB)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "Failed to change order partyB",})
		return 
	}

	c.JSON(http.StatusOK, "Successfully update order status.")
	c.String(200, "\n")
}

func SettleOrder(c *gin.Context) {
	var err error
	orderid := c.Param("orderid")
	_, err = fabric.Contract.SubmitTransaction("SettleOrder", orderid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to settle order: " + err.Error()})
		return
	}

	err = mysql.UpdateOrderStatus(orderid, "FINISHED")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "Failed to change order status",})
		return 
	}

	c.JSON(http.StatusOK, "Successfully settle order.")
	c.String(200, "\n")
}

func ListOrders(c *gin.Context) {
	var orders []mysql.MysqlOrder
	var err error
	var orderstatus, ordertype string

	orderstatus = c.Param("status")
	ordertype = c.Param("type")

	if !UserLogged(){
		c.JSON(http.StatusOK, gin.H{"message": "NO user logged in, cannot create an order",})
		return 
	}
	
	var userinfo *jwt.LoginUser
	userinfo, err = jwt.ParseToken(CURRENT_USER)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "Failed to parse token" + CURRENT_USER,})
		return
	}

	userid := userinfo.UserID


	if ordertype != "ALL"{
		ordertype = userid
	}

	orders, err = mysql.ReturnOrders(orderstatus, ordertype)
	if (err != nil) {
		c.JSON(http.StatusOK, "Failed to get orders")
	}
	c.JSON(http.StatusOK, orders)
}

func ListUserOrders(c *gin.Context) {
	var orders []mysql.MysqlOrder
	var err error
	if !UserLogged(){
		c.JSON(http.StatusOK, gin.H{"message": "NO user logged in, cannot create a order",})
		return 
	}
	
	var userinfo *jwt.LoginUser
	userinfo, err = jwt.ParseToken(CURRENT_USER)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "Failed to parse token" + CURRENT_USER,})
		return
	}


	userid := userinfo.UserID

	orders, err = mysql.ReturnOrders("ALL", userid)
	if (err != nil) {
		c.JSON(http.StatusOK, "Failed to get orders")
	}
	c.JSON(http.StatusOK, orders)
}

func ListNewOrders(c *gin.Context) {
	var orders []mysql.MysqlOrder
	var err error
	orders, err = mysql.ReturnOrders("ALL", "ALL")
	if (err != nil) {
		c.JSON(http.StatusOK, "Failed to get orders")
	}
	n := len(orders)
    if n <= 5 {
        c.JSON(http.StatusOK, orders)
		return
	}
    c.JSON(http.StatusOK, orders[n-5:])
}
	