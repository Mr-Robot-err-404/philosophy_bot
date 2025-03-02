package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
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
	requestCredentials(config)
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

func saveCredentials(token *oauth2.Token) error {
	env, err := getEnvMap()
	if err != nil {
		env = make(map[string]string)
	}
	env["ACCESS_TOKEN"] = token.AccessToken
	env["REFRESH_TOKEN"] = token.RefreshToken

	err = godotenv.Write(env, "./.env")
	if err != nil {
		return err
	}
	return nil

}

func requestCredentials(config *oauth2.Config) {
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
	url := config.AuthCodeURL("state", oauth2.AccessTypeOffline)

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
		return
	}

	fmt.Println("token received, shutting down local server...")

	if err := srv.Shutdown(context.Background()); err != nil {
		log.Fatalf("Failed to shut down server: %v", err)
	}
	fmt.Println("-----------")
	fmt.Println("access token -> ", token.AccessToken)
	fmt.Println("refresh token -> ", token.RefreshToken)
	fmt.Println("-----------")
	err = saveCredentials(token)
	if err != nil {
		log.Fatal("failed to save credentials to env file")
	}
	fmt.Println("credentials saved")
}

func getCredentials() Credentials {
	access_token := os.Getenv("ACCESS_TOKEN")
	key := os.Getenv("QUOTE_API_KEY")
	bearer := os.Getenv("BEARER")
	return Credentials{key: key, access_token: access_token, bearer: bearer}
}
