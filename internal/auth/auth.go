package auth

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

var (
	clientAuthConfig *Config
)

type Config struct {
	KeycloakBaseUri     string
	KeycloakLoginConfig oauth2.Config
}

func buildAuthConfig(context context.Context) *Config {
	baseProviderUrl := "http://localhost:8080/realms/myrealm"
	provider, err := oidc.NewProvider(context, baseProviderUrl)
	if err != nil {
		panic("Could not load OIDC configuration: " + err.Error())
	}

	config := &Config{
		KeycloakLoginConfig: oauth2.Config{
			ClientID: "test-notes-cli",
			Endpoint: provider.Endpoint(),
			Scopes:   []string{"profile", "email", oidc.ScopeOpenID},
		},
		KeycloakBaseUri: baseProviderUrl,
		// KeycloakIDTokenVerifier: provider.Verifier(&oidc.Config{ClientID: AuthConfig.KeycloakLoginConfig.ClientID}),
	}
	return config

}

func InitializeAuth() {
	clientAuthConfig = buildAuthConfig(context.Background())
}

func Login() (*oauth2.Token, error) {
	ctx := context.Background()
	deviceAuth, err := clientAuthConfig.KeycloakLoginConfig.DeviceAuth(ctx)
	if err != nil {
		return nil, err
	}

	completeUrl := deviceAuth.VerificationURIComplete
	if completeUrl != "" {
		fmt.Printf("> Visit the following URL to complete login: %s\n", completeUrl)
	} else {
		fmt.Printf("> Visit the following URL and enter the device code to complete login: %s\n", deviceAuth.VerificationURI)
		fmt.Printf("> Code: %s\n", deviceAuth.UserCode)
	}
	fmt.Printf("\n> Waiting for login (expires at: %s)...\n", deviceAuth.Expiry.Local())

	token, err := clientAuthConfig.KeycloakLoginConfig.DeviceAccessToken(ctx, deviceAuth)
	if err != nil {
		return nil, err // TODO: Better error message here?
	}

	return token, nil
}
