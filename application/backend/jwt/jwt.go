package jwt

import (
	"time"

	"github.com/golang-jwt/jwt"
)

var TokenExpireDuration = time.Hour * 6 

func GenToken(userid string) (string, error) {	
	secret := []byte("test")
	tmp := loginUser{
		UserID:   userid,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(TokenExpireDuration).Unix(),
			Issuer:    "FabricServer",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, tmp)
	return token.SignedString(secret)
}
