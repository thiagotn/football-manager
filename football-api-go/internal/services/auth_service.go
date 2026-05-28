package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	twilioApi "github.com/twilio/twilio-go/rest/verify/v2"
	"golang.org/x/crypto/bcrypt"

	twilio "github.com/twilio/twilio-go"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/config"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
)

var whatsappRegex = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)

// ── Request / Response types ─────────────────────────────────────────────────

type LoginRequest struct {
	WhatsApp string `json:"whatsapp"`
	Password string `json:"password"`
}

type SendOTPRequest struct {
	WhatsApp string `json:"whatsapp"`
}

type SendOTPResponse struct {
	Status           string `json:"status"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

type VerifyOTPRequest struct {
	WhatsApp string `json:"whatsapp"`
	OTPCode  string `json:"otp_code"`
}

type VerifyOTPMeRequest struct {
	OTPCode string `json:"otp_code"`
}

type VerifyOTPResponse struct {
	OTPToken string `json:"otp_token"`
}

type RegisterRequest struct {
	Name     string  `json:"name"`
	Nickname *string `json:"nickname,omitempty"`
	WhatsApp string  `json:"whatsapp"`
	Password string  `json:"password"`
	OTPToken string  `json:"otp_token"`
}

type TokenResponse struct {
	AccessToken        string  `json:"access_token"`
	RefreshToken       string  `json:"refresh_token"`
	TokenType          string  `json:"token_type"`
	PlayerID           string  `json:"player_id"`
	Name               string  `json:"name"`
	Nickname           *string `json:"nickname,omitempty"`
	Role               string  `json:"role"`
	MustChangePassword bool    `json:"must_change_password"`
	AvatarURL          *string `json:"avatar_url,omitempty"`
	ChatEnabled        bool    `json:"chat_enabled"`
}

type PlayerResponse struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Nickname           *string   `json:"nickname,omitempty"`
	WhatsApp           string    `json:"whatsapp"`
	Role               string    `json:"role"`
	Active             bool      `json:"active"`
	MustChangePassword bool      `json:"must_change_password"`
	AvatarURL          *string   `json:"avatar_url,omitempty"`
	ChatEnabled        bool      `json:"chat_enabled"`
	CreatedAt          time.Time `json:"created_at"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

type ChangePasswordRequest struct {
	CurrentPassword *string `json:"current_password,omitempty"`
	NewPassword     string  `json:"new_password"`
	OTPToken        *string `json:"otp_token,omitempty"`
}

type ForgotPasswordResetRequest struct {
	WhatsApp    string `json:"whatsapp"`
	NewPassword string `json:"new_password"`
	OTPToken    string `json:"otp_token"`
}

// ── AuthService interface ────────────────────────────────────────────────────

type AuthService interface {
	Login(ctx context.Context, req LoginRequest) (*TokenResponse, error)
	SendOTP(ctx context.Context, req SendOTPRequest) (*SendOTPResponse, error)
	VerifyOTP(ctx context.Context, req VerifyOTPRequest) (*VerifyOTPResponse, error)
	Register(ctx context.Context, req RegisterRequest) (*TokenResponse, error)
	GetMe(ctx context.Context, playerID uuid.UUID) (*PlayerResponse, error)
	ForgotPasswordSendOTP(ctx context.Context, req SendOTPRequest) (*SendOTPResponse, error)
	ForgotPasswordVerifyOTP(ctx context.Context, req VerifyOTPRequest) (*VerifyOTPResponse, error)
	ForgotPasswordReset(ctx context.Context, req ForgotPasswordResetRequest) error
	SendOTPMe(ctx context.Context, playerID uuid.UUID) (*SendOTPResponse, error)
	VerifyOTPMe(ctx context.Context, playerID uuid.UUID, req VerifyOTPMeRequest) (*VerifyOTPResponse, error)
	ChangePassword(ctx context.Context, playerID uuid.UUID, req ChangePasswordRequest) error
	RefreshToken(ctx context.Context, req RefreshRequest) (*RefreshResponse, error)
	IssueTokenPairForPlayer(ctx context.Context, player *db.Player) (*TokenResponse, error)
}

// ── Implementation ───────────────────────────────────────────────────────────

type authService struct {
	pool   *pgxpool.Pool
	cfg    *config.Config
	twilio *twilio.RestClient
}

func NewAuthService(pool *pgxpool.Pool, cfg *config.Config) AuthService {
	var twilioClient *twilio.RestClient
	if cfg.TwilioAccountSID != "" && cfg.TwilioAuthToken != "" {
		twilioClient = twilio.NewRestClientWithParams(twilio.ClientParams{
			Username: cfg.TwilioAccountSID,
			Password: cfg.TwilioAuthToken,
		})
	}
	return &authService{pool: pool, cfg: cfg, twilio: twilioClient}
}

// Login authenticates a player and returns a token pair.
func (s *authService) Login(ctx context.Context, req LoginRequest) (*TokenResponse, error) {
	wa, err := normalizeWhatsApp(req.WhatsApp)
	if err != nil {
		return nil, err
	}

	player, err := db.GetPlayerByWhatsApp(ctx, s.pool, wa)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, apierror.Forbidden("invalid credentials")
		}
		return nil, fmt.Errorf("authService.Login: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(player.PasswordHash), []byte(req.Password)); err != nil {
		return nil, apierror.Forbidden("invalid credentials")
	}

	return s.issueTokenPair(ctx, player)
}

// SendOTP sends an OTP to a WhatsApp number for registration.
func (s *authService) SendOTP(ctx context.Context, req SendOTPRequest) (*SendOTPResponse, error) {
	wa, err := normalizeWhatsApp(req.WhatsApp)
	if err != nil {
		return nil, err
	}
	if err := s.sendOTPToNumber(wa); err != nil {
		return nil, fmt.Errorf("authService.SendOTP: %w", err)
	}
	return &SendOTPResponse{Status: "pending", ExpiresInSeconds: 600}, nil
}

// VerifyOTP verifies an OTP code and returns a short-lived otp_token.
func (s *authService) VerifyOTP(ctx context.Context, req VerifyOTPRequest) (*VerifyOTPResponse, error) {
	wa, err := normalizeWhatsApp(req.WhatsApp)
	if err != nil {
		return nil, err
	}
	if err := validateOTPCode(req.OTPCode); err != nil {
		return nil, err
	}
	ok, err := s.checkOTP(wa, req.OTPCode)
	if err != nil || !ok {
		return nil, apierror.Forbidden("invalid OTP code")
	}
	token, err := s.createOTPToken(wa)
	if err != nil {
		return nil, err
	}
	return &VerifyOTPResponse{OTPToken: token}, nil
}

// Register creates a new player account.
func (s *authService) Register(ctx context.Context, req RegisterRequest) (*TokenResponse, error) {
	if len(strings.TrimSpace(req.Name)) < 2 {
		return nil, apierror.Unprocessable("name must be at least 2 characters")
	}
	if len(req.Password) < 6 {
		return nil, apierror.Unprocessable("password must be at least 6 characters")
	}

	wa, err := normalizeWhatsApp(req.WhatsApp)
	if err != nil {
		return nil, err
	}

	// Validate otp_token
	otpWA, err := s.decodeOTPToken(req.OTPToken)
	if err != nil || otpWA != wa {
		return nil, apierror.Forbidden("invalid or expired OTP token")
	}

	// Check if WhatsApp is already registered
	existing, err := db.GetPlayerByWhatsApp(ctx, s.pool, wa)
	if err == nil && existing != nil {
		return nil, apierror.Conflict("whatsapp already registered")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("authService.Register bcrypt: %w", err)
	}

	var nickname *string
	if req.Nickname != nil {
		trimmed := strings.TrimSpace(*req.Nickname)
		if trimmed != "" {
			nickname = &trimmed
		}
	}

	player, err := db.CreatePlayer(ctx, s.pool, db.CreatePlayerArgs{
		Name:         strings.TrimSpace(req.Name),
		Nickname:     nickname,
		WhatsApp:     wa,
		PasswordHash: string(hash),
	})
	if err != nil {
		return nil, fmt.Errorf("authService.Register create: %w", err)
	}

	slog.Info("player registered", "player_id", player.ID, "whatsapp", wa)

	return s.issueTokenPair(ctx, player)
}

// GetMe returns the player profile.
func (s *authService) GetMe(ctx context.Context, playerID uuid.UUID) (*PlayerResponse, error) {
	player, err := db.GetPlayerByID(ctx, s.pool, playerID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, apierror.NotFound("player not found")
		}
		return nil, err
	}
	return playerToResponse(player), nil
}

// ForgotPasswordSendOTP sends an OTP to a registered WhatsApp number.
func (s *authService) ForgotPasswordSendOTP(ctx context.Context, req SendOTPRequest) (*SendOTPResponse, error) {
	wa, err := normalizeWhatsApp(req.WhatsApp)
	if err != nil {
		return nil, err
	}
	// Verify the player exists
	if _, err := db.GetPlayerByWhatsApp(ctx, s.pool, wa); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			// Don't reveal whether the number is registered
			return &SendOTPResponse{Status: "pending", ExpiresInSeconds: 600}, nil
		}
		return nil, err
	}
	if err := s.sendOTPToNumber(wa); err != nil {
		return nil, err
	}
	return &SendOTPResponse{Status: "pending", ExpiresInSeconds: 600}, nil
}

// ForgotPasswordVerifyOTP verifies an OTP for password reset.
func (s *authService) ForgotPasswordVerifyOTP(ctx context.Context, req VerifyOTPRequest) (*VerifyOTPResponse, error) {
	wa, err := normalizeWhatsApp(req.WhatsApp)
	if err != nil {
		return nil, err
	}
	if err := validateOTPCode(req.OTPCode); err != nil {
		return nil, err
	}
	ok, err := s.checkOTP(wa, req.OTPCode)
	if err != nil || !ok {
		return nil, apierror.Forbidden("invalid OTP code")
	}
	token, err := s.createOTPToken(wa)
	if err != nil {
		return nil, err
	}
	return &VerifyOTPResponse{OTPToken: token}, nil
}

// ForgotPasswordReset resets a password using an otp_token.
func (s *authService) ForgotPasswordReset(ctx context.Context, req ForgotPasswordResetRequest) error {
	if len(req.NewPassword) < 6 {
		return apierror.Unprocessable("password must be at least 6 characters")
	}
	wa, err := normalizeWhatsApp(req.WhatsApp)
	if err != nil {
		return err
	}
	otpWA, err := s.decodeOTPToken(req.OTPToken)
	if err != nil || otpWA != wa {
		return apierror.Forbidden("invalid or expired OTP token")
	}
	player, err := db.GetPlayerByWhatsApp(ctx, s.pool, wa)
	if err != nil {
		return apierror.NotFound("player not found")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 12)
	if err != nil {
		return err
	}
	if err := db.UpdatePlayerPassword(ctx, s.pool, player.ID, string(hash)); err != nil {
		return err
	}
	return db.RevokeAllRefreshTokensForPlayer(ctx, s.pool, player.ID)
}

// SendOTPMe sends an OTP to the authenticated player's own number.
func (s *authService) SendOTPMe(ctx context.Context, playerID uuid.UUID) (*SendOTPResponse, error) {
	player, err := db.GetPlayerByID(ctx, s.pool, playerID)
	if err != nil {
		return nil, apierror.NotFound("player not found")
	}
	if err := s.sendOTPToNumber(player.WhatsApp); err != nil {
		return nil, err
	}
	return &SendOTPResponse{Status: "pending", ExpiresInSeconds: 600}, nil
}

// VerifyOTPMe verifies an OTP for the authenticated player.
func (s *authService) VerifyOTPMe(ctx context.Context, playerID uuid.UUID, req VerifyOTPMeRequest) (*VerifyOTPResponse, error) {
	if err := validateOTPCode(req.OTPCode); err != nil {
		return nil, err
	}
	player, err := db.GetPlayerByID(ctx, s.pool, playerID)
	if err != nil {
		return nil, apierror.NotFound("player not found")
	}
	ok, err := s.checkOTP(player.WhatsApp, req.OTPCode)
	if err != nil || !ok {
		return nil, apierror.Forbidden("invalid OTP code")
	}
	token, err := s.createOTPToken(player.WhatsApp)
	if err != nil {
		return nil, err
	}
	return &VerifyOTPResponse{OTPToken: token}, nil
}

// ChangePassword updates the player's password.
// Requires either current_password (bcrypt) or otp_token (from VerifyOTPMe).
func (s *authService) ChangePassword(ctx context.Context, playerID uuid.UUID, req ChangePasswordRequest) error {
	if len(req.NewPassword) < 6 {
		return apierror.Unprocessable("new_password must be at least 6 characters")
	}
	if req.CurrentPassword == nil && req.OTPToken == nil {
		return apierror.Unprocessable("current_password or otp_token required")
	}

	player, err := db.GetPlayerByID(ctx, s.pool, playerID)
	if err != nil {
		return apierror.NotFound("player not found")
	}

	if req.OTPToken != nil {
		// Validate via OTP token
		otpWA, err := s.decodeOTPToken(*req.OTPToken)
		if err != nil || otpWA != player.WhatsApp {
			return apierror.Forbidden("invalid or expired OTP token")
		}
	} else {
		// Validate via current password
		if err := bcrypt.CompareHashAndPassword([]byte(player.PasswordHash), []byte(*req.CurrentPassword)); err != nil {
			return apierror.Forbidden("current password is incorrect")
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 12)
	if err != nil {
		return err
	}
	if err := db.UpdatePlayerPassword(ctx, s.pool, playerID, string(hash)); err != nil {
		return err
	}
	return db.RevokeAllRefreshTokensForPlayer(ctx, s.pool, playerID)
}

// RefreshToken rotates refresh tokens and returns a new token pair.
func (s *authService) RefreshToken(ctx context.Context, req RefreshRequest) (*RefreshResponse, error) {
	if req.RefreshToken == "" {
		return nil, apierror.Unprocessable("refresh_token is required")
	}

	tokenHash := db.HashToken(req.RefreshToken)
	rt, err := db.GetValidRefreshToken(ctx, s.pool, tokenHash)
	if err != nil {
		return nil, apierror.Forbidden("invalid or expired refresh token")
	}

	// Rotate: revoke old token
	if err := db.RevokeRefreshToken(ctx, s.pool, tokenHash); err != nil {
		return nil, err
	}

	player, err := db.GetPlayerByID(ctx, s.pool, rt.PlayerID)
	if err != nil {
		return nil, apierror.NotFound("player not found")
	}

	accessToken, err := s.createAccessToken(player.ID.String())
	if err != nil {
		return nil, err
	}

	newRefreshToken, newHash, err := generateRefreshToken()
	if err != nil {
		return nil, err
	}
	if err := db.CreateRefreshToken(ctx, s.pool, player.ID, newHash); err != nil {
		return nil, err
	}

	return &RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "bearer",
	}, nil
}

// ── Internal helpers ─────────────────────────────────────────────────────────

func (s *authService) issueTokenPair(ctx context.Context, player *db.Player) (*TokenResponse, error) {
	accessToken, err := s.createAccessToken(player.ID.String())
	if err != nil {
		return nil, err
	}

	refreshToken, refreshHash, err := generateRefreshToken()
	if err != nil {
		return nil, err
	}
	if err := db.CreateRefreshToken(ctx, s.pool, player.ID, refreshHash); err != nil {
		return nil, fmt.Errorf("issueTokenPair: %w", err)
	}

	return &TokenResponse{
		AccessToken:        accessToken,
		RefreshToken:       refreshToken,
		TokenType:          "bearer",
		PlayerID:           player.ID.String(),
		Name:               player.Name,
		Nickname:           player.Nickname,
		Role:               string(player.Role),
		MustChangePassword: player.MustChangePassword,
		AvatarURL:          player.AvatarURL,
		ChatEnabled:        player.ChatEnabled,
	}, nil
}

// IssueTokenPairForPlayer issues a token pair for an existing player.
// Used when a player joins a group via invite and needs to be authenticated.
func (s *authService) IssueTokenPairForPlayer(ctx context.Context, player *db.Player) (*TokenResponse, error) {
	return s.issueTokenPair(ctx, player)
}

func (s *authService) createAccessToken(subject string) (string, error) {
	exp := time.Duration(s.cfg.AccessTokenExpireMinutes) * time.Minute
	claims := jwt.RegisteredClaims{
		Subject:   subject,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(exp)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString([]byte(s.cfg.SecretKey))
}

func (s *authService) createOTPToken(whatsapp string) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   "otp:" + whatsapp,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString([]byte(s.cfg.SecretKey))
}

func (s *authService) decodeOTPToken(tokenStr string) (string, error) {
	tok, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{},
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(s.cfg.SecretKey), nil
		},
	)
	if err != nil || !tok.Valid {
		return "", errors.New("invalid token")
	}
	claims, ok := tok.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return "", errors.New("invalid claims")
	}
	sub, err := claims.GetSubject()
	if err != nil || !strings.HasPrefix(sub, "otp:") {
		return "", errors.New("not an OTP token")
	}
	return strings.TrimPrefix(sub, "otp:"), nil
}

func (s *authService) sendOTPToNumber(whatsapp string) error {
	if !s.cfg.OTPEnabled() && s.cfg.OTPBypassCode != "" {
		slog.Info("OTP bypass active — skipping Twilio", "whatsapp", whatsapp)
		return nil
	}
	if s.twilio == nil {
		return apierror.Internal("OTP service not configured")
	}
	params := &twilioApi.CreateVerificationParams{}
	params.SetTo(whatsapp)
	params.SetChannel("sms")
	_, err := s.twilio.VerifyV2.CreateVerification(s.cfg.TwilioVerifySID, params)
	return err
}

func (s *authService) checkOTP(whatsapp, code string) (bool, error) {
	if !s.cfg.OTPEnabled() && s.cfg.OTPBypassCode != "" {
		return code == s.cfg.OTPBypassCode, nil
	}
	if s.twilio == nil {
		return false, apierror.Internal("OTP service not configured")
	}
	params := &twilioApi.CreateVerificationCheckParams{}
	params.SetTo(whatsapp)
	params.SetCode(code)
	result, err := s.twilio.VerifyV2.CreateVerificationCheck(s.cfg.TwilioVerifySID, params)
	if err != nil {
		return false, nil
	}
	return result.Status != nil && *result.Status == "approved", nil
}

// normalizeWhatsApp strips formatting and validates E.164 format.
func normalizeWhatsApp(raw string) (string, error) {
	// Strip everything except digits and leading +
	var b strings.Builder
	for i, ch := range raw {
		if i == 0 && ch == '+' {
			b.WriteRune(ch)
			continue
		}
		if ch >= '0' && ch <= '9' {
			b.WriteRune(ch)
		}
	}
	wa := b.String()
	if !whatsappRegex.MatchString(wa) {
		return "", apierror.Unprocessable("invalid whatsapp format (E.164 required, e.g. +5511999990000)")
	}
	return wa, nil
}

func validateOTPCode(code string) error {
	if len(code) != 6 {
		return apierror.Unprocessable("otp_code must be exactly 6 digits")
	}
	for _, ch := range code {
		if ch < '0' || ch > '9' {
			return apierror.Unprocessable("otp_code must contain only digits")
		}
	}
	return nil
}

func generateRefreshToken() (token, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generateRefreshToken: %w", err)
	}
	token = base64.URLEncoding.EncodeToString(b)
	hash = db.HashToken(token)
	return token, hash, nil
}

func playerToResponse(p *db.Player) *PlayerResponse {
	return &PlayerResponse{
		ID:                 p.ID.String(),
		Name:               p.Name,
		Nickname:           p.Nickname,
		WhatsApp:           p.WhatsApp,
		Role:               string(p.Role),
		Active:             p.Active,
		MustChangePassword: p.MustChangePassword,
		AvatarURL:          p.AvatarURL,
		ChatEnabled:        p.ChatEnabled,
		CreatedAt:          p.CreatedAt,
	}
}
