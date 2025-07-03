package middleware

import (
	"strings"

	"github.com/Dziqha/TurboGo/core"
	"github.com/golang-jwt/jwt/v5"
	"github.com/valyala/fasthttp"
)

func AuthJWT(secret string) core.Handler {
	return func(c *core.Context) {
		authHeader := string(c.Ctx.Request.Header.Peek("Authorization"))
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.Ctx.SetStatusCode(fasthttp.StatusUnauthorized)
			c.JSON(401, map[string]any{
				"error":   "unauthorized",
				"message": "missing or invalid authorization header",
			})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.Ctx.SetStatusCode(fasthttp.StatusUnauthorized)
			c.JSON(401, map[string]any{
				"error":   "unauthorized",
				"message": "invalid token",
			})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			for _, v := range claims {
				if str, ok := v.(string); ok {
					c.SetSession("user", str)
					break 
				}
			}
		}

		c.Next()
	}
}
