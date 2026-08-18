package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/agnos-assessment/agnos-backend/internal/service"
	"github.com/agnos-assessment/agnos-backend/pkg/response"
)

// StaffClaimsKey is the gin context key under which parsed StaffClaims are stored.
const StaffClaimsKey = "staff_claims"

// AuthRequired returns a Gin middleware that validates a Bearer JWT in the
// Authorization header. On success it stores *service.StaffClaims in the
// context under StaffClaimsKey and calls c.Next(). On failure it aborts with 401.
func AuthRequired(jwtSecret string) gin.HandlerFunc {
	secret := []byte(jwtSecret)
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized,
				response.NewError("UNAUTHORIZED", "missing or invalid authorization header"))
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.ParseWithClaims(tokenStr, &service.StaffClaims{}, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return secret, nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized,
				response.NewError("UNAUTHORIZED", "invalid or expired token"))
			return
		}

		claims, ok := token.Claims.(*service.StaffClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized,
				response.NewError("UNAUTHORIZED", "invalid token claims"))
			return
		}

		c.Set(StaffClaimsKey, claims)
		c.Next()
	}
}
