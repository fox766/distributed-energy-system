package main

import (
	"gin-app/api"
	"gin-app/fabric"

	"github.com/gin-gonic/gin"
)

func main() {
	fabric.InitFabric()

	r := gin.Default()
	api.RegisterAssetRoutes(r)

	r.Run("0.0.0.0:8080")
}
