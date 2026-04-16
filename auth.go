package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/justjanne/seafile-notifications/config"
	log "github.com/sirupsen/logrus"
)

type JwtClaims struct {
	ValidToken bool
	Exp        int64  `json:"exp"`
	RepoID     string `json:"repo_id"`
	UserName   string `json:"username"`
	jwt.RegisteredClaims
}

func (claims *JwtClaims) Valid() error {
	if !claims.ValidToken {
		return fmt.Errorf("jwt token is not valid")
	}
	if claims.Exp <= time.Now().Unix() {
		return fmt.Errorf("jwt token has expired")
	}
	return nil
}

func (claims *JwtClaims) ValidForScope(scope string) error {
	if err := claims.Valid(); err != nil {
		return err
	}
	if claims.RepoID != scope {
		return fmt.Errorf("jwt scope does not match: %s != %s", scope, claims.RepoID)
	}
	return nil
}

func ParseToken(config config.AppConfig, value string) (JwtClaims, error) {
	var claims JwtClaims
	if value == "" {
		return claims, fmt.Errorf("could not parse jwt token: no token provided")
	}
	token, err := jwt.ParseWithClaims(value, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.PrivateKey), nil
	})
	if err != nil {
		return claims, fmt.Errorf("could not parse jwt token: %w", err)
	}
	claims.ValidToken = token.Valid
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
	if claims, err := ParseTokenWithScope(state.Config, tokenString, repoID); err != nil {
		log.Warnf("could not parse jwt: %v\n", err)
		return "", -1, false
	} else if err = claims.ValidForScope(repoID); err != nil {
		return "", -1, false
	} else {
		return claims.UserName, claims.Exp, true
	}
}

func (state *AppContext) checkAuthToken(tokenString string) bool {
	if claims, err := ParseToken(state.Config, tokenString); err != nil {
		log.Warnf("could not parse jwt: %v\n", err)
		return false
	} else if err = claims.Valid(); err != nil {
		return false
	} else {
		return true
	}
}

func getAuthorizationToken(h http.Header) string {
	auth := h.Get("Authorization")
	splitResult := strings.Split(auth, " ")
	if len(splitResult) > 1 {
		return splitResult[1]
	}
	return ""
}
