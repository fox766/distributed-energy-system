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



