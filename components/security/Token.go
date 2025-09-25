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

// Claims defines the JWT claims structure
type Claims struct {
	UserID   string `json:"UserID"` 
	IsAdmin  bool
	IsActive bool
	jwt.StandardClaims
}

// GenerateToken generates a signed JWT token
func (c *Claims) GenerateToken() (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	tokenString, err := token.SignedString([]byte(config.JWTKey.GetStringValue()))
	if err != nil {
		log.GetLogger().Error(err.Error())
		return "", errors.NewHTTPError("unable to generate token", http.StatusInternalServerError)
	}
	return tokenString, nil
}

// ExtractUserIDFromToken extracts the UUID from the JWT token in the request header
func ExtractUserIDFromToken(r *http.Request) (uuid.UUID, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return uuid.Nil, errors.NewHTTPError("token not provided", http.StatusUnauthorized)
	}

	tokenStr := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	fmt.Println("Extracted Token: ", tokenStr)

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.NewHTTPError(errors.ErrorCodeInternalError, http.StatusInternalServerError)
		}
		return []byte(config.JWTKey.GetStringValue()), nil
	})
	if err != nil {
		return uuid.Nil, errors.NewHTTPError(err.Error(), http.StatusUnauthorized)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		id, ok := claims["UserID"].(string)
		if !ok || id == "" {
			return uuid.Nil, errors.NewHTTPError("Invalid user ID in token", http.StatusUnauthorized)
		}

		userID, err := util.ParseUUID(id)
		if err != nil {
			return uuid.Nil, err
		}
		return userID, nil
	}

	return uuid.Nil, errors.NewHTTPError("Invalid token.", http.StatusUnauthorized)
}
