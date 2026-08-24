package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"context"
	//"crypto/rand"
	//"encoding/base64"
	"errors"
	"fmt"
	"os"
	"time"

	"institution-os/pkg/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrAccountLocked   = errors.New("account is temporarily locked")
	ErrAccountInactive = errors.New("account is inactive")
	ErrMFARequired     = errors.New("MFA verification required")
	ErrInvalidMFACode  = errors.New("invalid MFA code")
	ErrPasswordReused  = errors.New("password was recently used")
	ErrInvalidToken    = errors.New("invalid or expired token")
	ErrTokenExpired    = errors.New("token has expired")
)

// Service implements all authentication business logic.
type Service struct {
	repo       *Repository
	cfg        *config.Config
	privateKey interface{} // *rsa.PrivateKey
	publicKey  interface{} // *rsa.PublicKey
}

// NewService creates a new auth service.
func NewService(repo *Repository, cfg *config.Config) (*Service, error) {
	svc := &Service{repo: repo, cfg: cfg}
	if err := svc.loadKeys(); err != nil {
		return nil, err
	}
	return svc, nil
}

func (s *Service) loadKeys() error {
	privBytes, err := os.ReadFile(s.cfg.JWTPrivateKeyPath)
	if err != nil {
		return fmt.Errorf("read private key: %w", err)
	}
	privKey, err := jwt.ParseRSAPrivateKeyFromPEM(privBytes)
	if err != nil {
		return fmt.Errorf("parse private key: %w", err)
	}
	s.privateKey = privKey

	pubBytes, err := os.ReadFile(s.cfg.JWTPublicKeyPath)
	if err != nil {
		return fmt.Errorf("read public key: %w", err)
	}
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(pubBytes)
	if err != nil {
		return fmt.Errorf("parse public key: %w", err)
	}
	s.publicKey = pubKey

	return nil
}

// Login authenticates a user and returns tokens.
func (s *Service) Login(ctx context.Context, institutionID uuid.UUID, req *LoginRequest, userAgent, ipAddress string) (*LoginResponse, error) {
	// Try lookup by email, fall back to phone
	var user *User
	var passwordHash string
	var err error

	user, passwordHash, err = s.repo.GetUserByEmail(ctx, institutionID, req.Email)
	if errors.Is(err, ErrUserNotFound) {
		// Try phone
		user, passwordHash, err = s.repo.GetUserByPhone(ctx, institutionID, req.Email)
	}
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// Check account active
	if !user.IsActive {
		return nil, ErrAccountInactive
	}

	// Check account locked
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return nil, ErrAccountLocked
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		_ = s.repo.IncrementLoginAttempts(ctx, user.ID)
		return nil, ErrInvalidCredentials
	}

	// Reset login attempts
	_ = s.repo.ResetLoginAttempts(ctx, user.ID)

	// MFA check
	if user.MFAEnabled {
		mfaToken, err := s.generateMFAToken(user.ID)
		if err != nil {
			return nil, err
		}
		return &LoginResponse{
			MFARequired: true,
			MFAToken:    mfaToken,
		}, nil
	}

	return s.issueTokens(ctx, user, userAgent, ipAddress)
}

// VerifyMFA verifies a TOTP code and issues tokens.
func (s *Service) VerifyMFA(ctx context.Context, institutionID uuid.UUID, req *MFAVerifyRequest, userAgent, ipAddress string) (*LoginResponse, error) {
	userID, err := s.validateMFAToken(req.MFAToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	user, err := s.repo.GetUserByID(ctx, institutionID, userID)
	if err != nil {
		return nil, err
	}

	secret, err := s.repo.GetMFASecret(ctx, userID)
	if err != nil {
		return nil, err
	}

	if !totp.Validate(req.TOTPCode, secret) {
		return nil, ErrInvalidMFACode
	}

	return s.issueTokens(ctx, user, userAgent, ipAddress)
}

// issueTokens generates access + refresh tokens for an authenticated user.
func (s *Service) issueTokens(ctx context.Context, user *User, userAgent, ipAddress string) (*LoginResponse, error) {
	perms, err := s.repo.GetPermissionsForRole(ctx, user.RoleID)
	if err != nil {
		return nil, fmt.Errorf("get permissions: %w", err)
	}

	accessToken, err := s.generateAccessToken(user, perms)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, tokenHash, err := s.generateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	expiresAt := time.Now().Add(s.cfg.JWTRefreshTTL)
	if err := s.repo.StoreRefreshToken(ctx, user.InstitutionID, user.ID, tokenHash, userAgent, ipAddress, expiresAt); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	profile := &UserProfile{
		ID:              user.ID,
		InstitutionID:   user.InstitutionID,
		Role:            user.RoleSlug,
		FullName:        user.FirstName + " " + user.LastName,
		Email:           user.Email,
		Phone:           user.Phone,
		ProfilePhotoURL: user.ProfilePhotoURL,
		Permissions:     perms,
	}

	return &LoginResponse{
		AccessToken:        accessToken,
		TokenType:          "Bearer",
		ExpiresIn:          int(s.cfg.JWTAccessTTL.Seconds()),
		MustChangePassword: user.MustChangePassword,
		User:               profile,
		RefreshToken:       refreshToken,
	}, nil
}

func hashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// jwtClaims is the JWT payload struct.
type jwtClaims struct {
	jwt.RegisteredClaims
	UserID        string   `json:"uid"`
	InstitutionID string   `json:"iid"`
	Role          string   `json:"role"`
	Permissions   []string `json:"perms"`
}

func (s *Service) generateAccessToken(user *User, perms []string) (string, error) {
	now := time.Now()
	claims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.JWTAccessTTL)),
			ID:        uuid.New().String(), // jti for blacklisting
		},
		UserID:        user.ID.String(),
		InstitutionID: user.InstitutionID.String(),
		Role:          user.RoleSlug,
		Permissions:   perms,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(s.privateKey)
}

func (s *Service) generateRefreshToken() (token, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	token = base64.URLEncoding.EncodeToString(b)
	// Refresh tokens are already 256 bits of randomness — bcrypt's salting is
	// for low-entropy secrets like passwords and actively breaks lookup here
	// (bcrypt is non-deterministic; you can't recompute the same hash to find
	// the row by exact match). SHA-256 is the correct, standard choice for
	// hashing an already-high-entropy token.
	sum := sha256.Sum256([]byte(token))
	hash = hex.EncodeToString(sum[:])
	return token, hash, nil
}

func (s *Service) generateMFAToken(userID uuid.UUID) (string, error) {
	// Short-lived JWT with only user ID, no permissions
	claims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
		},
		UserID: userID.String(),
		Role:   "mfa_pending",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(s.privateKey)
}

func (s *Service) validateMFAToken(tokenStr string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(t *jwt.Token) (interface{}, error) {
		return s.publicKey, nil
	})
	if err != nil || !token.Valid {
		return uuid.Nil, ErrInvalidToken
	}
	claims := token.Claims.(*jwtClaims)
	if claims.Role != "mfa_pending" {
		return uuid.Nil, ErrInvalidToken
	}
	return uuid.Parse(claims.UserID)
}

// ParseAccessToken validates and parses an access token, returning Claims.
func (s *Service) ParseAccessToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.publicKey, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}
	if !token.Valid {
		return nil, ErrInvalidToken
	}
	jc, ok := token.Claims.(*jwtClaims)
	if !ok {
		return nil, ErrInvalidToken
	}
	return &Claims{
		UserID:        jc.UserID,
		InstitutionID: jc.InstitutionID,
		Role:          jc.Role,
		Permissions:   jc.Permissions,
	}, nil
}

// ChangePassword validates the current password and sets a new one.
func (s *Service) ChangePassword(ctx context.Context, institutionID, userID uuid.UUID, req *ChangePasswordRequest) error {
	currentHash, err := s.repo.GetPasswordHash(ctx, userID)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(req.CurrentPassword)); err != nil {
		return ErrInvalidCredentials
	}

	// Check history
	history, err := s.repo.GetPasswordHistory(ctx, userID, 5)
	if err != nil {
		return err
	}
	for _, h := range history {
		if bcrypt.CompareHashAndPassword([]byte(h), []byte(req.NewPassword)) == nil {
			return ErrPasswordReused
		}
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 12)
	if err != nil {
		return err
	}

	if err := s.repo.UpdatePasswordHash(ctx, userID, string(newHash)); err != nil {
		return err
	}

	return s.repo.SavePasswordHistory(ctx, institutionID, userID, string(newHash))
}

// HashPassword hashes a plain-text password using bcrypt.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (s *Service) RefreshToken(ctx context.Context, rawToken, userAgent, ipAddress string) (*LoginResponse, error) {
	if rawToken == "" {
		return nil, ErrInvalidToken
	}
	hash := hashRefreshToken(rawToken)

	userID, institutionID, expiresAt, err := s.repo.GetRefreshToken(ctx, hash)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if expiresAt.Before(time.Now()) {
		_ = s.repo.RevokeRefreshToken(ctx, hash)
		return nil, ErrTokenExpired
	}

	user, err := s.repo.GetUserByID(ctx, *institutionID, *userID)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if !user.IsActive {
		return nil, ErrAccountInactive
	}

	// Rotate: invalidate the presented token before issuing a new one.
	if err := s.repo.RevokeRefreshToken(ctx, hash); err != nil {
		return nil, fmt.Errorf("revoke old refresh token: %w", err)
	}

	return s.issueTokens(ctx, user, userAgent, ipAddress)
}
