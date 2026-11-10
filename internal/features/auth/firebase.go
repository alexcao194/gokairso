package auth

import (
	"context"
	"fmt"

	"github.com/alexcao194/kairso/pkg/firebase"
)

type firebaseVerifierAdapter struct {
	verifier firebase.Verifier
}

func NewFirebaseVerifier(ctx context.Context, credentialsPath string) (FirebaseVerifier, error) {
	if credentialsPath == "" {
		return nil, ErrFirebaseNotConfigured
	}
	v, err := firebase.NewVerifier(ctx, credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("initialize firebase verifier: %w", err)
	}
	return &firebaseVerifierAdapter{verifier: v}, nil
}

func (a *firebaseVerifierAdapter) VerifyIDToken(ctx context.Context, idToken string) (*FirebaseToken, error) {
	token, err := a.verifier.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, err
	}
	return mapFirebaseToken(token), nil
}
