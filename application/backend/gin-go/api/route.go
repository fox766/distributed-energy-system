package api

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/cors"
	"time"
)

func RegisterAssetRoutes(r *gin.Engine) {
	// 添加CORS中间件
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // 允许所有来源
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	
	r.GET("/init", Init)
	r.GET("/register/:username/:password", RegisterUser)
	r.GET("/login/:username/:password", Login)
	r.GET("/logout", Logout)
	r.GET("/getuser/:userid", GetUser)
	r.GET("/updateuser/:userid/:available/:balance", UpdateUser)
	r.GET("/getorder/:orderid", GetOrder)
	r.GET("/createorder/:amount", CreateOrder)
	r.GET("/matchorder/:orderid", MatchOrder)
	r.GET("/settleorder/:orderid", SettleOrder)
	r.GET("/getallorders/:status/:type", ListOrders)
	r.GET("/getuserorders", ListUserOrders)
	r.GET("/getsystemstatus", ReturnSystemStatus)
	r.GET("/listneworders", ListNewOrders)
}