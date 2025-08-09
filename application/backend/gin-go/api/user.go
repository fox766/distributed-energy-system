package api

import (
	//"fmt"
	"strconv"
	"net/http"
	"encoding/json"

	"backend/gin-go/fabric"
	"backend/mysql"
	"backend/jwt"

	"github.com/gin-gonic/gin"
)


var CURRENT_USER string

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
	userrole := "User"
	availableStr := strconv.FormatFloat(0.0, 'f', 2, 64)
	balanceStr := strconv.FormatFloat(balance, 'f', 2, 64) 

	_, err := fabric.Contract.SubmitTransaction("RegisterUser", userid, username, userrole, availableStr, balanceStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, "Successfully Register." + "Your userid is " + userid)

}

func GetUser(c *gin.Context) {
	var userid string
	userid = c.Param("userid")
	result, err := fabric.Contract.EvaluateTransaction("GetUser", userid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read user: " + err.Error()})
		return
	}

	var user User
	json.Unmarshal(result, &user)
	c.JSON(http.StatusOK, user)
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

	CURRENT_USER, err = jwt.GenToken(userid)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "failed to generate token.",})
		return
	}

	c.JSON(http.StatusOK, "Successfully login. Hi, " + userid)
}

func Logout(c *gin.Context) {
	if CURRENT_USER == "" {
		c.JSON(http.StatusOK, gin.H{"message": "NO user logged in, no need to log out.",})
	}
	CURRENT_USER = ""
	c.JSON(http.StatusOK, "Successfully logout, Bye.")
}


var CNT_USER int

func genuserid() (userid string){
	var number string

	CNT_USER += 1
	number = strconv.Itoa(CNT_USER)
	userid = "energy_user_"+number

	return userid
}

func UserInit() {
	CNT_USER = 0
	CURRENT_USER = ""
}

func UserLogged() bool{
	return CURRENT_USER != ""
}