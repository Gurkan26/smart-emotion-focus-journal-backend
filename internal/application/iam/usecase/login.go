package usecase

import (
	"context"

	"github.com/gurkanfikretgunak/masterfabric-go/internal/application/iam/dto"
	"github.com/gurkanfikretgunak/masterfabric-go/internal/domain/iam/repository"
	"github.com/gurkanfikretgunak/masterfabric-go/internal/domain/iam/service"
	domainErr "github.com/gurkanfikretgunak/masterfabric-go/internal/shared/errors"
)

// LoginUseCase handles user authentication.
type LoginUseCase struct {
	userRepo repository.UserRepository
	auth     service.AuthService
}

// NewLoginUseCase creates a new LoginUseCase.
func NewLoginUseCase(userRepo repository.UserRepository, auth service.AuthService) *LoginUseCase {
	return &LoginUseCase{userRepo: userRepo, auth: auth}
}

// Execute authenticates a user and returns a JWT token.
func (uc *LoginUseCase) Execute(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error) {
	user, err := uc.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrUnauthorized, "invalid credentials", nil)
	}

	if !user.IsActive() {
		return nil, domainErr.New(domainErr.ErrForbidden, "account is not active", nil)
	}

	if err := uc.auth.VerifyPassword(user.PasswordHash, req.Password); err != nil {
		return nil, err
	}

	token, err := uc.auth.GenerateToken(ctx, service.TokenClaims{
		UserID: user.ID,
		Email:  user.Email,
	})
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to generate token", err)
	}

	return &dto.LoginResponse{
		Token: token,
		User: dto.UserInfo{
			ID:        user.ID,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Status:    string(user.Status),
			CreatedAt: user.CreatedAt,
		},
	}, nil
}
