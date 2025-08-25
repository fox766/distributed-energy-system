package api

import (
	"fmt"
	"encoding/json"
	"net/http"
	//"time"
	
	"backend/gin-go/fabric"
	"backend/mysql"

	"github.com/gin-gonic/gin"
)


func EnergyInit() error{
	_, err := fabric.Contract.SubmitTransaction("Init")
	if err != nil {
		return fmt.Errorf("failed to Init EnergyStatus.")
	}
	return nil 
}


func Init(c *gin.Context) {
	_, err := fabric.Contract.SubmitTransaction("Init")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to Initialize asset: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, "Successfully Initialization.")
	//c.String(http.StatusOK, "Successfully Initialization.")
	c.String(http.StatusOK, "\n")
}

func ReturnSystemStatus(c *gin.Context) {
	var systemvar SystemStatus
	result, err := fabric.Contract.EvaluateTransaction("GetEnergyStatus")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read energystatus: " + err.Error()})
		return
	}

	var energy EnergyStatus
	json.Unmarshal(result, &energy)

	systemvar.EnergyPrice = energy.EnergyPrice
	_, systemvar.UserNum = mysql.ReturnOrderNum()
	_, systemvar.OrderNum = mysql.ReturnUserNum()
	if systemvar.UserNum == -1 || systemvar.OrderNum == -1 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read num from db: "})
	}
	c.JSON(http.StatusOK, systemvar)
}



