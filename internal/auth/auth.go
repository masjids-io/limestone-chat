package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type AuthContextKey string

const UserIDContextKey AuthContextKey = "userID"

type UnprotectedRoute struct {
	Path   string
	Method string
}

func GenerateJWT(userID uuid.UUID) (string, string, error) {
	accessSecret := os.Getenv("ACCESS_SECRET")
	refreshSecret := os.Getenv("REFRESH_SECRET")
	accessExpiration := os.Getenv("ACCESS_EXPIRATION")
	refreshExpiration := os.Getenv("REFRESH_EXPIRATION")

	if accessSecret == "" || refreshSecret == "" || accessExpiration == "" || refreshExpiration == "" {
		return "", "", fmt.Errorf("JWT secrets or expiration not set in environment variables")
	}

	accessExpMinutes, err := strconv.Atoi(accessExpiration)
	if err != nil {
		accessExpMinutes = 60
	}
	refreshExpHours, err := strconv.Atoi(refreshExpiration)
	if err != nil {
		refreshExpHours = 24 * 7
	}

	now := time.Now()
	accessClaims := jwt.MapClaims{
		"user_id": userID.String(),
		"exp":     now.Add(time.Minute * time.Duration(accessExpMinutes)).Unix(),
		"iat":     now.Unix(),
	}

	refreshClaims := jwt.MapClaims{
		"user_id": userID.String(),
		"exp":     now.Add(time.Hour * time.Duration(refreshExpHours)).Unix(),
		"iat":     now.Unix(),
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)

	accessString, err := accessToken.SignedString([]byte(accessSecret))
	if err != nil {
		return "", "", fmt.Errorf("failed to sign access token: %w", err)
	}

	refreshString, err := refreshToken.SignedString([]byte(refreshSecret))
	if err != nil {
		return "", "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return accessString, refreshString, nil
}

func GetUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	val := ctx.Value(UserIDContextKey)
	userIDStr, ok := val.(string)
	if !ok {
		return uuid.Nil, false
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, false
	}
	return userID, true
}

func VerifyJWTForWebSocket(r *http.Request) (uuid.UUID, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return uuid.Nil, fmt.Errorf("authorization header required")
	}

	bearerToken := strings.Split(authHeader, " ")
	if len(bearerToken) != 2 || strings.ToLower(bearerToken[0]) != "bearer" {
		return uuid.Nil, fmt.Errorf("invalid authorization header format")
	}

	tokenString := bearerToken[1]
	accessSecret := os.Getenv("ACCESS_SECRET")
	if accessSecret == "" {
		log.Println("ACCESS_SECRET not set for WebSocket authentication")
		return uuid.Nil, fmt.Errorf("server configuration error: ACCESS_SECRET not set")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(accessSecret), nil
	})

	if err != nil {
		log.Printf("Error parsing token (WebSocket): %v", err)
		return uuid.Nil, fmt.Errorf("invalid token: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userIDStr, ok := claims["user_id"].(string)
		if !ok {
			return uuid.Nil, fmt.Errorf("invalid token claims: user_id not found or invalid type")
		}
		parsedUserID, err := uuid.Parse(userIDStr)
		if err != nil {
			return uuid.Nil, fmt.Errorf("invalid user ID in token claims: %w", err)
		}
		return parsedUserID, nil
	}

	return uuid.Nil, fmt.Errorf("invalid token or claims")
}
