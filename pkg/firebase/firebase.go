package firebase

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

type Token struct {
	UID            string
	Email          string
	EmailVerified  bool
	Name           string
	Picture        string
	SignInProvider string
}

type Verifier interface {
	VerifyIDToken(ctx context.Context, idToken string) (*Token, error)
}

type firebaseAuthClient interface {
	VerifyIDToken(ctx context.Context, idToken string) (*firebaseauth.Token, error)
}

type verifier struct {
	client firebaseAuthClient
}

func NewVerifier(ctx context.Context, credentialsPath string) (Verifier, error) {
	if credentialsPath == "" {
		return nil, fmt.Errorf("credentials path is required")
	}
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(credentialsPath))
	if err != nil {
		return nil, fmt.Errorf("initialize firebase app: %w", err)
	}
	client, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialize firebase auth client: %w", err)
	}
	return &verifier{client: client}, nil
}

func (v *verifier) VerifyIDToken(ctx context.Context, idToken string) (*Token, error) {
	token, err := v.client.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, err
	}
	return tokenFromVerifiedToken(token), nil
}

func tokenFromVerifiedToken(token *firebaseauth.Token) *Token {
	if token == nil {
		return nil
	}
	email, _ := token.Claims["email"].(string)
	emailVerified, _ := token.Claims["email_verified"].(bool)
	name, _ := token.Claims["name"].(string)
	picture, _ := token.Claims["picture"].(string)
	provider := ""
	if firebaseClaim, ok := token.Claims["firebase"].(map[string]any); ok {
		provider, _ = firebaseClaim["sign_in_provider"].(string)
	}
	return &Token{
		UID:            token.UID,
		Email:          email,
		EmailVerified:  emailVerified,
		Name:           name,
		Picture:        picture,
		SignInProvider: provider,
	}
}
