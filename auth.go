package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/justjanne/seafile-notifications/config"
)

type JwtClaims struct {
	Exp      int64  `json:"exp"`
	RepoID   string `json:"repo_id"`
	UserName string `json:"username"`
	jwt.RegisteredClaims
}

func (*JwtClaims) Valid() error {
	return nil
}

func ParseToken(config config.AppConfig, value string) (JwtClaims, error) {
	var claims JwtClaims
	if value == "" {
		return claims, fmt.Errorf("could not parse jwt token: no token provided")
	}
	token, err := jwt.ParseWithClaims(value, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.PrivateKey), nil
	})
	if err != nil {
		return claims, fmt.Errorf("could not parse jwt token: %w", err)
	}
	if !token.Valid {
		return claims, fmt.Errorf("jwt token is not valid")
	}
	if claims.Exp <= time.Now().Unix() {
		return claims, fmt.Errorf("jwt token has expired")
	}
	return claims, nil
}

func ParseTokenWithScope(config config.AppConfig, value, repoId string) (JwtClaims, error) {
	if claims, err := ParseToken(config, value); err != nil {
		return claims, err
	} else if claims.RepoID != repoId {
		return claims, fmt.Errorf("jwt scope does not match: %s != %s", repoId, claims.RepoID)
	} else {
		return claims, nil
	}
}

func (state *AppContext) checkToken(tokenString, repoID string) (string, int64, bool) {
	claims, err := ParseTokenWithScope(state.Config, tokenString, repoID)
	if err != nil {
		return "", -1, false
	}

	return claims.UserName, claims.Exp, true
}

func (state *AppContext) checkAuthToken(tokenString string) bool {
	if _, err := ParseToken(state.Config, tokenString); err != nil {
		return false
	}

	return true
}

func getAuthorizationToken(h http.Header) string {
	auth := h.Get("Authorization")
	splitResult := strings.Split(auth, " ")
	if len(splitResult) > 1 {
		return splitResult[1]
	}
	return ""
}
