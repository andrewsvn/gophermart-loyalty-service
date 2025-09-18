package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/andrewsvn/gophermart-ls/internal/auth"
	"github.com/andrewsvn/gophermart-ls/internal/logging"
	"github.com/andrewsvn/gophermart-ls/internal/model"
	"github.com/andrewsvn/gophermart-ls/internal/repository"
	"github.com/andrewsvn/gophermart-ls/internal/utils"
	"go.uber.org/zap"
)

type UserService struct {
	repoFacade *repository.Facade
	idp        *auth.IdentityProvider
	logger     *zap.SugaredLogger
}

var (
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrWrongLoginPassword = errors.New("wrong combination of login and password used")
)

func NewUserService(rf *repository.Facade, idp *auth.IdentityProvider, l *zap.Logger) *UserService {
	return &UserService{
		repoFacade: rf,
		idp:        idp,
		logger:     logging.ComponentLogger(l, "user-management"),
	}
}

func (s *UserService) RegisterUser(ctx context.Context, login, password string) error {
	exists, err := s.repoFacade.CheckUserExistsByLogin(ctx, login)
	if err != nil {
		return fmt.Errorf("unable to check if user exists: %w", err)
	}
	if exists {
		return fmt.Errorf("%w: %s", ErrUserAlreadyExists, login)
	}

	authHash := utils.LoginPassHash(login, password)
	userID, err := s.repoFacade.CreateUser(ctx, login, authHash)
	if err != nil {
		return fmt.Errorf("unable to create user: %w", err)
	}

	s.logger.Infow("new user created",
		"ID", userID,
		"login", login,
	)
	return nil
}

// LoginUser validates user data and generates new access token to be provided in an Authorization header
func (s *UserService) LoginUser(ctx context.Context, login, password string) (*model.AuthorizationResult, error) {
	user, err := s.repoFacade.GetUserByLogin(ctx, login)
	if err != nil {
		return nil, fmt.Errorf("unable to get user: %w", err)
	}
	if user == nil {
		return nil, ErrWrongLoginPassword
	}

	authHash := utils.LoginPassHash(login, password)
	if user.AuthHash != authHash {
		return nil, ErrWrongLoginPassword
	}

	token, err := s.idp.GenerateAccessToken(user.ID, user.AuthHash)
	if err != nil {
		return nil, fmt.Errorf("unable to generate access token: %w", err)
	}
	_, err = s.repoFacade.UpdateUserLoginTS(ctx, user.ID)
	if err != nil {
		s.logger.Warnw("unable to update user login", "error", err)
	}

	return &model.AuthorizationResult{
		AccessToken: token,
	}, nil
}
