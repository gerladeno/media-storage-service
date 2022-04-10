package rest

import (
	"context"
	"crypto/rsa"
	"errors"
	"net/http"
	"strings"

	"github.com/gerladeno/authorization-service/pkg/common"
	"github.com/golang-jwt/jwt"
)

type Claims struct {
	jwt.StandardClaims
	ID string `json:"id"`
}

type idType string

const idKey idType = `userID`

func (h *handler) auth(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeErrResponse(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		headerParts := strings.Split(authHeader, " ")
		if len(headerParts) != 2 {
			writeErrResponse(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if headerParts[0] != "Bearer" {
			writeErrResponse(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		id, err := parseToken(headerParts[1], h.key)
		switch {
		case err == nil:
		case errors.Is(err, common.ErrInvalidAccessToken):
			writeErrResponse(w, "Unauthorized", http.StatusUnauthorized)
			return
		default:
			h.log.Warnf("err parsing token: %v", err)
			writeErrResponse(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		r = r.WithContext(context.WithValue(r.Context(), idKey, id))
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func parseToken(accessToken string, key *rsa.PublicKey) (string, error) {
	token, err := jwt.ParseWithClaims(accessToken, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, common.ErrInvalidSigningMethod
		}
		return &key, nil
	})
	if err != nil {
		return "", err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims.ID, nil
	}
	return "", common.ErrInvalidAccessToken
}
