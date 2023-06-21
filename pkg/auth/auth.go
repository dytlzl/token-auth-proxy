package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const ProxyAuthorizationHeaderName = "Proxy-Authorization"

func InjectToken(r *http.Request, token string) {
	r.Header.Add(ProxyAuthorizationHeaderName, fmt.Sprintf("Bearer %s", token))
}

func ExtractToken(r *http.Request) (string, error) {
	splitValue := strings.Split(r.Header.Get(ProxyAuthorizationHeaderName), " ")
	r.Header.Del(ProxyAuthorizationHeaderName)
	if len(splitValue) != 2 {
		return "", fmt.Errorf("invalid %s header", ProxyAuthorizationHeaderName)
	}
	token := splitValue[1]
	return token, nil
}

type Authorizer interface {
	Authorize(ctx context.Context, token string) error
}

type SingleTokenAuthorizer struct {
	Token string
}

func (a SingleTokenAuthorizer) Authorize(ctx context.Context, token string) error {
	if token != a.Token {
		return errors.New("invalid token")
	}
	return nil
}
