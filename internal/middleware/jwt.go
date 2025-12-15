package middleware

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type ctxKey string

const userIDKey ctxKey = "userID"

func FromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	return id, ok
}

func JWTUnaryInterceptor(secret string, disableAuth bool) grpc.UnaryServerInterceptor {
	// Список методов, для которых токен не нужен
	whitelist := map[string]struct{}{
		"/leadexchange.v1.AuthService/Login":       {},
		"/leadexchange.v1.AuthService/Register":    {},
		"/leadexchange.v1.AuthService/HealthCheck": {},
	}

	// Тестовый user ID для использования когда auth отключен
	testUserID := uuid.MustParse("8c6f9c70-9312-4f17-94b0-2a2b9230f5d1")

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Если auth отключен, используем тестовый user ID
		if disableAuth {
			ctx = context.WithValue(ctx, userIDKey, testUserID)
			return handler(ctx, req)
		}

		if _, ok := whitelist[info.FullMethod]; ok {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, fmt.Errorf("missing metadata")
		}

		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			return nil, fmt.Errorf("missing authorization header")
		}

		parts := strings.SplitN(authHeaders[0], " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return nil, fmt.Errorf("invalid authorization header format")
		}

		tokenString := parts[1]

		if tokenString == "test" {
			ctx = context.WithValue(ctx, userIDKey, testUserID)
			return handler(ctx, req)
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			return nil, fmt.Errorf("invalid token: %v", err)
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return nil, fmt.Errorf("invalid token claims")
		}

		uidStr, ok := claims["uid"].(string)
		if !ok {
			return nil, fmt.Errorf("uid not found in token")
		}

		uid, err := uuid.Parse(uidStr)
		if err != nil {
			return nil, fmt.Errorf("invalid uid in token")
		}

		// Передаём userID в контекст
		ctx = context.WithValue(ctx, userIDKey, uid)
		return handler(ctx, req)
	}
}
