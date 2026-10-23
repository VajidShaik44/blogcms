package services

import (
	"errors"
	"testing"

	"backend/internal/config"
	"backend/internal/models"
	"backend/internal/repositories"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(user *models.User) error {
	return m.Called(user).Error(0)
}

func (m *MockUserRepository) GetByID(id uint) (*models.User, error) {
	args := m.Called(id)
	user, _ := args.Get(0).(*models.User)
	return user, args.Error(1)
}

func (m *MockUserRepository) GetByUsername(username string) (*models.User, error) {
	args := m.Called(username)
	user, _ := args.Get(0).(*models.User)
	return user, args.Error(1)
}

func (m *MockUserRepository) GetByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	user, _ := args.Get(0).(*models.User)
	return user, args.Error(1)
}

func (m *MockUserRepository) Update(user *models.User) error {
	return m.Called(user).Error(0)
}

func (m *MockUserRepository) Delete(id uint) error {
	return m.Called(id).Error(0)
}

func (m *MockUserRepository) List(page, perPage int) ([]models.User, int64, error) {
	args := m.Called(page, perPage)
	users, _ := args.Get(0).([]models.User)
	return users, args.Get(1).(int64), args.Error(2)
}

type MockJWTService struct {
	mock.Mock
}

func (m *MockJWTService) GenerateTokenPair(user *models.User) (*models.AuthResponse, error) {
	args := m.Called(user)
	response, _ := args.Get(0).(*models.AuthResponse)
	return response, args.Error(1)
}

func (m *MockJWTService) ValidateAccessToken(token string) (*models.JWTClaims, error) {
	args := m.Called(token)
	claims, _ := args.Get(0).(*models.JWTClaims)
	return claims, args.Error(1)
}

func (m *MockJWTService) ValidateRefreshToken(token string) (*models.JWTClaims, error) {
	args := m.Called(token)
	claims, _ := args.Get(0).(*models.JWTClaims)
	return claims, args.Error(1)
}

func (m *MockJWTService) RefreshAccessToken(token string) (*models.RefreshTokenResponse, error) {
	args := m.Called(token)
	response, _ := args.Get(0).(*models.RefreshTokenResponse)
	return response, args.Error(1)
}

func (m *MockJWTService) RevokeRefreshToken(token string) error {
	return m.Called(token).Error(0)
}

func (m *MockJWTService) RevokeAllUserTokens(userID uint) error {
	return m.Called(userID).Error(0)
}

func (m *MockJWTService) HashPassword(password string) (string, error) {
	args := m.Called(password)
	return args.String(0), args.Error(1)
}

func (m *MockJWTService) CheckPassword(password, hash string) bool {
	return m.Called(password, hash).Bool(0)
}

var (
	_ repositories.UserRepository = (*MockUserRepository)(nil)
	_ JWTService                 = (*MockJWTService)(nil)
)

func TestAuthServiceRegister(t *testing.T) {
	t.Run("registers a user and removes the password from the response", func(t *testing.T) {
		userRepo := new(MockUserRepository)
		jwtService := new(MockJWTService)
		service := NewAuthService(userRepo, jwtService, &config.Config{})

		userRepo.On("GetByUsername", "vajid").Return(nil, gorm.ErrRecordNotFound).Once()
		userRepo.On("GetByEmail", "vajid@example.com").Return(nil, gorm.ErrRecordNotFound).Once()
		jwtService.On("HashPassword", "password123").Return("hashed-password", nil).Once()
		userRepo.On("Create", mock.MatchedBy(func(user *models.User) bool {
			return user.Username == "vajid" &&
				user.Name == "Vajid" &&
				user.Email == "vajid@example.com" &&
				user.Password == "hashed-password" &&
				user.Role == "author"
		})).Run(func(args mock.Arguments) {
			args.Get(0).(*models.User).ID = 42
		}).Return(nil).Once()

		user, err := service.Register(&models.RegisterRequest{
			Username: "vajid",
			Name:     "Vajid",
			Email:    "vajid@example.com",
			Password: "password123",
		})

		require.NoError(t, err)
		require.Equal(t, uint(42), user.ID)
		require.Equal(t, "author", user.Role)
		require.Empty(t, user.Password)
		userRepo.AssertExpectations(t)
		jwtService.AssertExpectations(t)
	})

	t.Run("rejects an existing username", func(t *testing.T) {
		userRepo := new(MockUserRepository)
		service := NewAuthService(userRepo, new(MockJWTService), &config.Config{})
		userRepo.On("GetByUsername", "vajid").Return(&models.User{ID: 1}, nil).Once()

		user, err := service.Register(&models.RegisterRequest{Username: "vajid"})

		require.Nil(t, user)
		require.EqualError(t, err, "username already exists")
		userRepo.AssertExpectations(t)
	})

	t.Run("rejects an existing email", func(t *testing.T) {
		userRepo := new(MockUserRepository)
		service := NewAuthService(userRepo, new(MockJWTService), &config.Config{})
		userRepo.On("GetByUsername", "vajid").Return(nil, gorm.ErrRecordNotFound).Once()
		userRepo.On("GetByEmail", "vajid@example.com").Return(&models.User{ID: 1}, nil).Once()

		user, err := service.Register(&models.RegisterRequest{
			Username: "vajid",
			Email:    "vajid@example.com",
		})

		require.Nil(t, user)
		require.EqualError(t, err, "email already exists")
		userRepo.AssertExpectations(t)
	})
}

func TestAuthServiceLogin(t *testing.T) {
	t.Run("logs in with valid credentials and removes the password", func(t *testing.T) {
		userRepo := new(MockUserRepository)
		jwtService := new(MockJWTService)
		service := NewAuthService(userRepo, jwtService, &config.Config{})
		user := &models.User{
			ID:       42,
			Username: "vajid",
			Name:     "Vajid",
			Email:    "vajid@example.com",
			Password: "hashed-password",
			Role:     "author",
		}
		response := &models.AuthResponse{
			AccessToken:  "access",
			RefreshToken: "refresh",
			User:         *user,
		}

		userRepo.On("GetByEmail", user.Email).Return(user, nil).Once()
		jwtService.On("CheckPassword", "password123", "hashed-password").Return(true).Once()
		jwtService.On("GenerateTokenPair", user).Return(response, nil).Once()

		result, err := service.Login(&models.LoginRequest{
			Email:    user.Email,
			Password: "password123",
		})

		require.NoError(t, err)
		require.Equal(t, "access", result.AccessToken)
		require.Equal(t, "refresh", result.RefreshToken)
		require.Equal(t, "Vajid", result.User.Name)
		require.Empty(t, result.User.Password)
		userRepo.AssertExpectations(t)
		jwtService.AssertExpectations(t)
	})

	t.Run("rejects unknown email", func(t *testing.T) {
		userRepo := new(MockUserRepository)
		service := NewAuthService(userRepo, new(MockJWTService), &config.Config{})
		userRepo.On("GetByEmail", "missing@example.com").Return(nil, gorm.ErrRecordNotFound).Once()

		response, err := service.Login(&models.LoginRequest{Email: "missing@example.com"})

		require.Nil(t, response)
		require.EqualError(t, err, "invalid email or password")
		userRepo.AssertExpectations(t)
	})

	t.Run("rejects an incorrect password", func(t *testing.T) {
		userRepo := new(MockUserRepository)
		jwtService := new(MockJWTService)
		service := NewAuthService(userRepo, jwtService, &config.Config{})
		user := &models.User{Email: "vajid@example.com", Password: "hashed-password"}
		userRepo.On("GetByEmail", user.Email).Return(user, nil).Once()
		jwtService.On("CheckPassword", "wrong-password", "hashed-password").Return(false).Once()

		response, err := service.Login(&models.LoginRequest{
			Email:    user.Email,
			Password: "wrong-password",
		})

		require.Nil(t, response)
		require.EqualError(t, err, "invalid email or password")
		userRepo.AssertExpectations(t)
		jwtService.AssertExpectations(t)
	})

	t.Run("returns an error when repository lookup fails", func(t *testing.T) {
		userRepo := new(MockUserRepository)
		service := NewAuthService(userRepo, new(MockJWTService), &config.Config{})
		userRepo.On("GetByEmail", "vajid@example.com").Return(nil, errors.New("database unavailable")).Once()

		response, err := service.Login(&models.LoginRequest{Email: "vajid@example.com"})

		require.Nil(t, response)
		require.EqualError(t, err, "authentication failed")
		userRepo.AssertExpectations(t)
	})
}
