package jwt

import (
	"github.com/golang-jwt/jwt"
)

type LoginUser struct {
	UserID     string
	jwt.StandardClaims
}