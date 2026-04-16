package main

import (
	"net/http"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

func (state *AppContext) checkToken(tokenString, repoID string) (string, int64, bool) {
	if len(tokenString) == 0 {
		return "", -1, false
	}
	claims := new(myClaims)
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(state.Config.PrivateKey), nil
	})
	if err != nil {
		return "", -1, false
	}

	if !token.Valid {
		return "", -1, false
	}

	now := time.Now()
	if claims.RepoID != repoID || claims.Exp <= now.Unix() {
		return "", -1, false
	}

	return claims.UserName, claims.Exp, true
}

func getAuthorizationToken(h http.Header) string {
	auth := h.Get("Authorization")
	splitResult := strings.Split(auth, " ")
	if len(splitResult) > 1 {
		return splitResult[1]
	}
	return ""
}

func (state *AppContext) checkAuthToken(tokenString string) bool {
	if len(tokenString) == 0 {
		return false
	}
	claims := new(myClaims)
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(state.Config.PrivateKey), nil
	})
	if err != nil {
		return false
	}

	if !token.Valid {
		return false
	}

	now := time.Now()

	return claims.Exp > now.Unix()
}
