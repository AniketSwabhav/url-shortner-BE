package security

import (
	"fmt"
	"net/http"
	"strings"
	"url-shortner-be/components/config"
	"url-shortner-be/components/errors"
	"url-shortner-be/components/log"
	"url-shortner-be/components/util"

	"github.com/golang-jwt/jwt"
	uuid "github.com/satori/go.uuid"
)

type Claims struct {
	UserID   uuid.UUID
	IsAdmin  bool
	IsActive bool
	jwt.StandardClaims
}

func (c *Claims) GenerateToken() (string, error) {
	// NewWithClaims returns token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)

	// access token string based on token
	tokenString, err := token.SignedString([]byte(config.JWTKey.GetStringValue()))
	if err != nil {
		log.GetLogger().Error(err.Error())
		return "", errors.NewHTTPError("unable to generate token", http.StatusInternalServerError)
	}
	return tokenString, nil
}

func ExtractUserIDFromToken(r *http.Request) (uuid.UUID, error) {

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return uuid.Nil, errors.NewHTTPError("token not provided", http.StatusUnauthorized)
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	tokenStr = strings.TrimSpace(tokenStr)

	fmt.Println("Extracted Token: ", tokenStr)

	token, err := jwt.Parse(tokenStr,
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.NewHTTPError(errors.ErrorCodeInternalError, http.StatusInternalServerError)
			}
			return []byte(config.JWTKey.GetStringValue()), nil
		})
	if err != nil {
		return uuid.Nil, errors.NewHTTPError(err.Error(), http.StatusUnauthorized)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		id, _ := claims["UserID"].(string)
		userID, err := util.ParseUUID(id)
		if err != nil {
			return uuid.Nil, err
		}
		return userID, nil
	}

	return uuid.Nil, errors.NewHTTPError("Invalid token.", http.StatusUnauthorized)
}
