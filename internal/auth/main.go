package auth

import (
	"fmt"
	"net/http"
	"strings"
)

func GetBearerToken(headers http.Header) (string, error) {
	s := headers.Get("Authorization")
	err := fmt.Errorf("Invalid Auth: %s", s)

	if len(s) == 0 {
		return "", err
	}
	s = strings.TrimSpace(s)
	sub := "Bearer "
	bearer := strings.Contains(s, sub)

	if !bearer || len(sub) == len(s) {
		return "", err
	}
	token := s[len(sub):]
	return token, nil
}
