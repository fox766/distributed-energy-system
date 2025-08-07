package api

import (
	
	"github.com/gin-gonic/gin"
)


func RegisterAssetRoutes(r *gin.Engine) {
	r.GET("/init", InitLedger)
	r.GET("/asset/:id", ReadAsset)
	r.GET("/register/:username/:password", RegisterUser)
	r.GET("/login/:username/:password", Login)
	r.GET("/logout", Logout)
}