// Package jwt provides JWT generation, parsing, and an HTTP middleware
// to enforce Bearer-token authentication
package jwt

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	userIDKey contextKey = "user_id"
	loginKey  contextKey = "login"
)

// UserIDFromContext returns the authenticated user's id from ctx
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok
}

// LoginFromContext returns the authenticated user's login from ctx
func LoginFromContext(ctx context.Context) (string, bool) {
	login, ok := ctx.Value(loginKey).(string)
	return login, ok
}

// ContextWithUserID returns a copy of ctx carrying the given user id
func ContextWithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// ContextWithLogin returns a copy of ctx carrying the given login
func ContextWithLogin(ctx context.Context, login string) context.Context {
	return context.WithValue(ctx, loginKey, login)
}

// Claims is the custom JWT claim payload, embedding standard registered claims
type Claims struct {
	UserID string `json:"user_id"`
	Login  string `json:"login"`
	jwt.RegisteredClaims
}

// GenerateToken issues a signed JWT containing user_id and login
func GenerateToken(userID, login string, secret []byte) (string, error) {
	claims := Claims{
		UserID: userID,
		Login:  login,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ParseToken validates a signed token string and returns its claims
func ParseToken(tokenString string, secret []byte) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// JWT returns a middleware that validates the Authorization Bearer token
// and stores the resulting user_id/login in the request context
func JWT(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing token", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "invalid token format", http.StatusUnauthorized)
				return
			}

			claims, err := ParseToken(parts[1], secret)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			ctx := ContextWithLogin(r.Context(), claims.Login)
			ctx = ContextWithUserID(ctx, claims.UserID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
