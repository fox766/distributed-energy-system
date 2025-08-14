package api

import (
	//"encoding/json"
	"net/http"
	//"time"
	
	"backend/gin-go/fabric"

	"github.com/gin-gonic/gin"
)





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
	systemvar.EnergyPrice = 1.0
	systemvar.UserNum = 2
	systemvar.OrderNum = 1
	c.JSON(http.StatusOK, systemvar)
}



