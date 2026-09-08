package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func tokensPath() string {
	if path := os.Getenv("TOKENS_PATH"); path != "" {
		return path
	}
	return "./tokens.json"
}

func loadTokens() (Tokens, error) {
	var tokens Tokens

	data, err := os.ReadFile(tokensPath())
	if err != nil {
		return tokens, err
	}
	if err := json.Unmarshal(data, &tokens); err != nil {
		return tokens, fmt.Errorf("malformed token file %q: %w", tokensPath(), err)
	}
	return tokens, nil
}

func saveTokens(tokens Tokens) error {
	path := tokensPath()

	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tokens-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func saveAccessToken(access_token string) error {
	tokens, err := loadTokens()
	if err != nil {
		return err
	}
	tokens.AccessToken = access_token
	return saveTokens(tokens)
}

func saveRefreshToken(refresh_token string) error {
	tokens, err := loadTokens()
	if err != nil {
		return err
	}
	tokens.RefreshToken = refresh_token
	return saveTokens(tokens)
}
