package main

import (
	"fmt"
	"backend/gin-go/api"
	"backend/gin-go/fabric"

	"github.com/gin-gonic/gin"
	"backend/mysql"
)

func main() {
	fabric.InitFabric()

	r := gin.Default()
	api.RegisterAssetRoutes(r)
	// 用户初始化
	api.UserInit()
	api.OrderInit()
	api.EnergyInit()

	// 初始化数据库
	if err := mysql.Initmysql(); err != nil {
		fmt.Printf("init mysql failed,err:%v\n", err)
	}

	r.Run("0.0.0.0:8080")
}
