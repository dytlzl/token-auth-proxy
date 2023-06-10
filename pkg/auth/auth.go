package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const ProxyAuthorizationHeaderName = "Proxy-Authorization"

func SetToken(r *http.Request, token string) {
	r.Header.Add(ProxyAuthorizationHeaderName, fmt.Sprintf("Bearer %s", token))
}

func Authorize(r *http.Request) error {
	splitValue := strings.Split(r.Header.Get(ProxyAuthorizationHeaderName), " ")
	r.Header.Del(ProxyAuthorizationHeaderName)
	if len(splitValue) != 2 {
		return fmt.Errorf("invalid %s header", ProxyAuthorizationHeaderName)
	}
	token := splitValue[1]
	return verifyToken(token)
}

func verifyToken(token string) error {
	if token != "nekot" {
		return errors.New("invalid token")
	}
	return nil
}
