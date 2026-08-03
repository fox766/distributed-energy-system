package middleware

import (
	"net/http"
	"strings"
	"time"

	"backend/config"
	"backend/model"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

// GenerateToken creates a signed JWT for the given user.
func GenerateToken(cfg *config.Config, userID, userRole string) (string, error) {
	claims := jwt.MapClaims{
		"uid":  userID,
		"role": userRole,
		"exp":  time.Now().Add(6 * time.Hour).Unix(),
		"iss":  "energy-system",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

// ParseToken validates and extracts claims from a JWT string.
func ParseToken(cfg *config.Config, tokenStr string) (*model.Claims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		uid, _ := claims["uid"].(string)
		role, _ := claims["role"].(string)
		return &model.Claims{UserID: uid, UserRole: role}, nil
	}
	return nil, jwt.ErrSignatureInvalid
}

// AuthRequired is a Gin middleware that rejects requests without a valid JWT.
func AuthRequired(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.ErrorResponse{Error: "missing or invalid Authorization header"})
			return
		}
		claims, err := ParseToken(cfg, strings.TrimPrefix(auth, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.ErrorResponse{Error: "invalid or expired token"})
			return
		}
		c.Set("userID", claims.UserID)
		c.Set("userRole", claims.UserRole)
		c.Next()
	}
}

// GetUserID extracts the authenticated user's ID from the Gin context.
func GetUserID(c *gin.Context) string {
	id, _ := c.Get("userID")
	if s, ok := id.(string); ok {
		return s
	}
	return ""
}

// GetUserRole extracts the authenticated user's role from the Gin context.
func GetUserRole(c *gin.Context) string {
	role, _ := c.Get("userRole")
	if s, ok := role.(string); ok {
		return s
	}
	return ""
}

// AdminRequired is a Gin middleware that rejects non-admin requests.
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := GetUserRole(c)
		if role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, model.ErrorResponse{Error: "admin role required"})
			return
		}
		c.Next()
	}
}

// ProducerRequired is a Gin middleware that allows only PRODUCER and admin roles.
func ProducerRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := GetUserRole(c)
		if role != "PRODUCER" && role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, model.ErrorResponse{Error: "producer role required"})
			return
		}
		c.Next()
	}
}
