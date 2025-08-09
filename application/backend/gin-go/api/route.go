package api

import (
	
	"github.com/gin-gonic/gin"
)


func RegisterAssetRoutes(r *gin.Engine) {
	r.GET("/register/:username/:password", RegisterUser)
	r.GET("/login/:username/:password", Login)
	r.GET("/logout", Logout)
	r.GET("/getuser/:userid", GetUser)
	r.GET("/createorder/:amount", CreateOrder)
}