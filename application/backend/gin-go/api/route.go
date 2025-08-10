package api

import (
	
	"github.com/gin-gonic/gin"
)


func RegisterAssetRoutes(r *gin.Engine) {
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
}