package service

import (
	"context"

	"github.com/ekhrunov/messenger/server/internal/repository"
)

func ResolveCurrentUserID(
	ctx context.Context,
	tokenUser TokenUser,
	userRepo repository.UserRepository,
	oidcProviderRepo repository.OIDCProviderRepository,
) (string, error) {
	if tokenUser.Subject == "" || tokenUser.Provider == "" {
		return "", ErrChatAccessDenied
	}

	provider, err := oidcProviderRepo.GetByName(ctx, tokenUser.Provider)
	if err != nil {
		return "", ErrChatAccessDenied
	}

	users, err := userRepo.List(ctx, repository.UserFilter{
		Subject:        tokenUser.Subject,
		OIDCProviderID: provider.ID,
	})
	if err != nil {
		return "", err
	}
	if len(users) == 0 {
		return "", ErrChatAccessDenied
	}

	return users[0].ID, nil
}

func requireCurrentUserMatches(
	ctx context.Context,
	tokenUser TokenUser,
	userID string,
	userRepo repository.UserRepository,
	oidcProviderRepo repository.OIDCProviderRepository,
	accessDenied error,
) error {
	currentUserID, err := ResolveCurrentUserID(ctx, tokenUser, userRepo, oidcProviderRepo)
	if err != nil {
		return err
	}
	if userID != currentUserID {
		return accessDenied
	}
	return nil
}
