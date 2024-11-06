package auth

import (
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
	"mrshanahan.com/notes-term/internal/paths"
)

// var (
// 	Cache *TokenCache // = newTokenCache()
// )

func LoadToken() (*oauth2.Token, error) {
	cacheDir, err := paths.EnsureLocalCacheFolder()
	if err != nil {
		return nil, err
	}
	cacheFile := filepath.Join(cacheDir, "token")

	var token *oauth2.Token
	if _, err = os.Stat(cacheFile); err != nil && os.IsNotExist(err) {
		slog.Info("no token file")
		return &oauth2.Token{}, nil
	}

	f, err := os.Open(cacheFile)
	if err != nil {
		return nil, err
	}

	bytes, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(bytes, token)
	if err != nil {
		return nil, err
	}

	return token, nil
}

func SaveToken(t *oauth2.Token) error {
	cacheDir, err := paths.EnsureLocalCacheFolder()
	if err != nil {
		return err
	}
	cacheFile := filepath.Join(cacheDir, "token")

	bytes, err := json.Marshal(t)
	if err != nil {
		return err
	}

	err = os.WriteFile(cacheFile, bytes, 0600)
	if err != nil {
		return err
	}

	return nil
}

func IsValid(t *oauth2.Token) bool {
	return t.AccessToken != "" && t.Expiry.After(time.Now())
}
