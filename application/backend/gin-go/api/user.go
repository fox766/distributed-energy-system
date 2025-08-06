package api

import (
	//"fmt"
	"strconv"
	"net/http"

	"backend/gin-go/fabric"
	"backend/mysql"

	"github.com/gin-gonic/gin"
)


func RegisterUser(c *gin.Context) {
	var userid, username, password string
	var balance float64

	userid = genuserid()

	username = c.Param("username")
	password = c.Param("password")
	
	if err := mysql.InsertUser(userid, username, password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register: " + err.Error()})
		return
	}

	balance = 0.0
	balanceStr := strconv.FormatFloat(balance, 'f', 2, 64) 

	_, err := fabric.Contract.SubmitTransaction("RegisterUser", userid, username, balanceStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, "Successfully Register." + "Your userid is " + userid)

}

func Login (c *gin.Context) {
	var err error
	var userid string

	username := c.Param("username")
	password := c.Param("password")

	userid, err = mysql.GetUserID(username)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "username not exists",})
		return
	}

	err = mysql.CheckPassword(username, password)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "wrong password.",})
		return
	}

	c.JSON(http.StatusOK, "Successfully login. Hi, " + userid)
}


var CNT_USER int

func genuserid() (userid string){
	var number string

	CNT_USER += 1
	number = strconv.Itoa(CNT_USER)
	userid = "energy_user_"+number

	return userid
}

func GenUseridInit() {
	CNT_USER = 0
}