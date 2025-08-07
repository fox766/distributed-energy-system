package jwt

import (
	"github.com/golang-jwt/jwt"
)

type loginUser struct {
	UserID     string
	jwt.StandardClaims
}