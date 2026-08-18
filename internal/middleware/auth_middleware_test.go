package middleware_test

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/agnos-assessment/agnos-backend/internal/middleware"
	"github.com/agnos-assessment/agnos-backend/internal/service"
)

const testSecret = "middleware-test-secret"

func newRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/protected", middleware.AuthRequired(testSecret), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func makeToken(secret string, staffID, hospitalID int64, role string, exp time.Time) string {
	claims := service.StaffClaims{
		StaffID:    staffID,
		HospitalID: hospitalID,
		Role:       role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(secret))
	return signed
}

func get(router *gin.Engine, authHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	token := makeToken(testSecret, 1, 10, "staff", time.Now().Add(time.Hour))
	w := get(newRouter(), "Bearer "+token)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAuthMiddleware_ClaimsStoredInContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	var captured *service.StaffClaims
	r.GET("/protected", middleware.AuthRequired(testSecret), func(c *gin.Context) {
		v, _ := c.Get(middleware.StaffClaimsKey)
		captured, _ = v.(*service.StaffClaims)
		c.Status(http.StatusOK)
	})

	token := makeToken(testSecret, 7, 3, "admin", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(httptest.NewRecorder(), req)

	if captured == nil {
		t.Fatal("claims not set in context")
	}
	if captured.StaffID != 7 {
		t.Errorf("expected staff_id 7, got %d", captured.StaffID)
	}
	if captured.HospitalID != 3 {
		t.Errorf("expected hospital_id 3, got %d", captured.HospitalID)
	}
	if captured.Role != "admin" {
		t.Errorf("expected role 'admin', got %q", captured.Role)
	}
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	w := get(newRouter(), "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_MissingBearerPrefix(t *testing.T) {
	token := makeToken(testSecret, 1, 1, "staff", time.Now().Add(time.Hour))
	w := get(newRouter(), token) // raw token without "Bearer " prefix
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	token := makeToken(testSecret, 1, 1, "staff", time.Now().Add(-time.Hour))
	w := get(newRouter(), "Bearer "+token)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_WrongSecret(t *testing.T) {
	token := makeToken("other-secret", 1, 1, "staff", time.Now().Add(time.Hour))
	w := get(newRouter(), "Bearer "+token)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_Malformed(t *testing.T) {
	w := get(newRouter(), "Bearer not.a.jwt")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_WrongSigningMethod(t *testing.T) {
	// Sign with RS256 (asymmetric) — middleware rejects non-HMAC methods.
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	claims := service.StaffClaims{
		StaffID: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, _ := token.SignedString(privKey)

	w := get(newRouter(), "Bearer "+signed)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
