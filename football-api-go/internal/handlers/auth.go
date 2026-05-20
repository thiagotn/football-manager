package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
	"github.com/thiagotn/football-manager/football-api-go/internal/services"
)

type authHandler struct {
	svc       services.AuthService
	loginRL   *middleware.LoginRateLimiter
}

func NewAuthHandler(svc services.AuthService, loginRL *middleware.LoginRateLimiter) *authHandler {
	return &authHandler{svc: svc, loginRL: loginRL}
}

// PublicRoutes returns routes that don't require authentication.
func (h *authHandler) PublicRoutes() http.Handler {
	r := chi.NewRouter()
	r.With(h.loginRL.Middleware).Post("/login", h.login)
	r.Post("/send-otp", h.sendOTP)
	r.Post("/verify-otp", h.verifyOTP)
	r.Post("/register", h.register)
	r.Post("/forgot-password/send-otp", h.forgotPasswordSendOTP)
	r.Post("/forgot-password/verify-otp", h.forgotPasswordVerifyOTP)
	r.Post("/forgot-password/reset", h.forgotPasswordReset)
	r.Post("/refresh", h.refresh)
	return r
}

// ProtectedRoutes returns routes that require a valid JWT.
func (h *authHandler) ProtectedRoutes() http.Handler {
	r := chi.NewRouter()
	r.Get("/me", h.getMe)
	r.Post("/send-otp/me", h.sendOTPMe)
	r.Post("/verify-otp/me", h.verifyOTPMe)
	r.Post("/change-password", h.changePassword)
	return r
}

// @Summary     Login
// @Tags        auth
// @Param       body body services.LoginRequest true "Credentials"
// @Success     200  {object} services.TokenResponse
// @Failure     403  {object} apierror.APIError
// @Failure     429  {object} apierror.APIError
// @Router      /auth/login [post]
func (h *authHandler) login(w http.ResponseWriter, r *http.Request) {
	var req services.LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}
	resp, err := h.svc.Login(r.Context(), req)
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, resp)
}

// @Summary     Send OTP (registration)
// @Tags        auth
// @Param       body body services.SendOTPRequest true "WhatsApp number"
// @Success     200  {object} services.SendOTPResponse
// @Router      /auth/send-otp [post]
func (h *authHandler) sendOTP(w http.ResponseWriter, r *http.Request) {
	var req services.SendOTPRequest
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}
	resp, err := h.svc.SendOTP(r.Context(), req)
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, resp)
}

// @Summary     Verify OTP (registration)
// @Tags        auth
// @Param       body body services.VerifyOTPRequest true "WhatsApp + OTP code"
// @Success     200  {object} services.VerifyOTPResponse
// @Router      /auth/verify-otp [post]
func (h *authHandler) verifyOTP(w http.ResponseWriter, r *http.Request) {
	var req services.VerifyOTPRequest
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}
	resp, err := h.svc.VerifyOTP(r.Context(), req)
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, resp)
}

// @Summary     Register
// @Tags        auth
// @Param       body body services.RegisterRequest true "Registration data"
// @Success     201  {object} services.TokenResponse
// @Failure     409  {object} apierror.APIError
// @Router      /auth/register [post]
func (h *authHandler) register(w http.ResponseWriter, r *http.Request) {
	var req services.RegisterRequest
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}
	resp, err := h.svc.Register(r.Context(), req)
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusCreated, resp)
}

// @Summary     Get current player
// @Tags        auth
// @Security    BearerAuth
// @Success     200  {object} services.PlayerResponse
// @Router      /auth/me [get]
func (h *authHandler) getMe(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	resp, err := h.svc.GetMe(r.Context(), player.ID)
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, resp)
}

func (h *authHandler) forgotPasswordSendOTP(w http.ResponseWriter, r *http.Request) {
	var req services.SendOTPRequest
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}
	resp, err := h.svc.ForgotPasswordSendOTP(r.Context(), req)
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, resp)
}

func (h *authHandler) forgotPasswordVerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req services.VerifyOTPRequest
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}
	resp, err := h.svc.ForgotPasswordVerifyOTP(r.Context(), req)
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, resp)
}

func (h *authHandler) forgotPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req services.ForgotPasswordResetRequest
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}
	if err := h.svc.ForgotPasswordReset(r.Context(), req); err != nil {
		renderError(w, err)
		return
	}
	noContent(w)
}

func (h *authHandler) sendOTPMe(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	resp, err := h.svc.SendOTPMe(r.Context(), player.ID)
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, resp)
}

func (h *authHandler) verifyOTPMe(w http.ResponseWriter, r *http.Request) {
	var req services.VerifyOTPMeRequest
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}
	player := middleware.PlayerFromCtx(r.Context())
	resp, err := h.svc.VerifyOTPMe(r.Context(), player.ID, req)
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, resp)
}

func (h *authHandler) changePassword(w http.ResponseWriter, r *http.Request) {
	var req services.ChangePasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}
	player := middleware.PlayerFromCtx(r.Context())
	if err := h.svc.ChangePassword(r.Context(), player.ID, req); err != nil {
		renderError(w, err)
		return
	}
	noContent(w)
}

func (h *authHandler) refresh(w http.ResponseWriter, r *http.Request) {
	var req services.RefreshRequest
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}
	resp, err := h.svc.RefreshToken(r.Context(), req)
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, resp)
}
