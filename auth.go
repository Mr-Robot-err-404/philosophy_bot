package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/youtube/v3"
)

func authenticate_account() error {
	config, err := extractGoogleConfig("client_secret.json")
	if err != nil {
		return err
	}
	access, refresh_token := requestCredentials(config)

	if refresh_token == "" {
		return fmt.Errorf("google returned no refresh token, revoke access at https://myaccount.google.com/permissions and retry")
	}
	if err := saveTokens(Tokens{AccessToken: access, RefreshToken: refresh_token}); err != nil {
		return err
	}
	printBreak()
	fmt.Printf("saved tokens -> %s\n", tokensPath())
	printBreak()

	return nil
}

func extractGoogleConfig(path string) (*oauth2.Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return &oauth2.Config{}, err
	}
	config, err := google.ConfigFromJSON(b, youtube.YoutubeReadonlyScope)
	if err != nil {
		return &oauth2.Config{}, err
	}
	config.Scopes = append(config.Scopes, youtube.YoutubeForceSslScope)
	return config, nil
}

func handleOauthCallback(ch chan<- string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if len(code) == 0 {
			http.Error(w, "No code was found in callback", http.StatusBadRequest)
			return
		}
		ch <- code
		html, err := os.ReadFile("./auth.html")
		if err != nil {
			w.WriteHeader(400)
			return
		}
		r.Header.Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write(html)
	}
}

func requestCredentials(config *oauth2.Config) (string, string) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	sslcli := &http.Client{Transport: tr}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, sslcli)

	ch := make(chan string)

	mux := http.NewServeMux()
	srv := &http.Server{Handler: mux, Addr: ":4545"}

	mux.HandleFunc("/auth/google/callback", handleOauthCallback(ch))

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()
	url := config.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	fmt.Printf("Your browser has been opened to visit::\n%s\n", url)

	if err := browser.OpenURL(url); err != nil {
		panic(fmt.Errorf("failed to open browser for authentication %s", err.Error()))
	}
	code := <-ch

	token, err := config.Exchange(context.Background(), code)
	if err != nil {
		log.Fatalf("Failed to exchange authorization code for token: %v", err)
	}

	if !token.Valid() {
		log.Fatalf("Can't get source information without accessToken: %v", err)
	}

	fmt.Println("token received, shutting down local server...")

	if err := srv.Shutdown(context.Background()); err != nil {
		log.Fatalf("Failed to shut down server: %v", err)
	}
	return token.AccessToken, token.RefreshToken
}

func getCredentials() Credentials {
	key := os.Getenv("QUOTE_API_KEY")
	bearer := os.Getenv("BEARER")

	tokens, err := loadTokens()
	if err != nil {
		if !os.IsNotExist(err) {
			log.Fatal(err)
		}
		tokens = Tokens{AccessToken: os.Getenv("ACCESS_TOKEN"), RefreshToken: os.Getenv("REFRESH_TOKEN")}

		if tokens.RefreshToken == "" {
			log.Fatalf("no tokens at %q and no REFRESH_TOKEN in env, run: ./bot -refresh", tokensPath())
		}
		if err := saveTokens(tokens); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("migrated tokens from env -> %s\n", tokensPath())
	}
	return Credentials{key: key, access_token: tokens.AccessToken, bearer: bearer, refresh_token: tokens.RefreshToken}
}
