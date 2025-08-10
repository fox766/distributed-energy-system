package jwt

import (
	"time"

	"github.com/golang-jwt/jwt"
)

var TokenExpireDuration = time.Hour * 6 

func GenToken(userid string) (string, error) {	
	secret := []byte("test")
	tmp := LoginUser{
		UserID:   userid,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(TokenExpireDuration).Unix(),
			Issuer:    "FabricServer",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, tmp)
	return token.SignedString(secret)
}

func ParseToken(tokenString string) (*LoginUser, error) {
	token, err := jwt.ParseWithClaims(tokenString, &LoginUser{}, func(token *jwt.Token) (i interface{}, err error) {
		return []byte("test"), nil
	})
	if err != nil {
		return nil, err
	}
	if userinfo, ok := token.Claims.(*LoginUser); ok && token.Valid {
		return userinfo, nil
	}
	return nil, err
}
