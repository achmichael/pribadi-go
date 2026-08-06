package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/achmichael/pribadi-go/internal/auth/domain"
	"github.com/achmichael/pribadi-go/internal/auth/usecase"
)

type AuthHandler struct {
	uc  *usecase.AuthUsecase
	val *validator.Validate
}

func NewAuthHandler(uc *usecase.AuthUsecase) *AuthHandler {
	return &AuthHandler{uc: uc, val: validator.New()}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}
	if err := h.val.Struct(req); err != nil {
		h.respondError(w, http.StatusUnprocessableEntity, "validation_failed", err.Error())
		return
	}

	err := h.uc.Register(r.Context(), req.FullName, req.Phone, req.Password)
	if err != nil {
		h.mapDomainError(w, err)
		return
	}

	h.respond(w, http.StatusCreated, map[string]string{"message": "OTP sent"})
}

func (h *AuthHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
    var req VerifyOTPReq
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}
    if err := h.val.Struct(req); err != nil {
		h.respondError(w, http.StatusUnprocessableEntity, "validation_failed", err.Error())
		return
	}
    
    err := h.uc.VerifyOTP(r.Context(), req.Phone, req.Code, domain.PurposeRegister)
    if err != nil {
		h.mapDomainError(w, err)
		return
	}
    
    h.respond(w, http.StatusOK, map[string]string{"message": "Phone verified successfully"})
}

func (h *AuthHandler) ResendOTP(w http.ResponseWriter, r *http.Request) {
    var req ResendOTPReq
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}
    if err := h.val.Struct(req); err != nil {
		h.respondError(w, http.StatusUnprocessableEntity, "validation_failed", err.Error())
		return
	}
    
    err := h.uc.ResendOTP(r.Context(), req.Phone, domain.PurposeRegister)
    if err != nil {
		h.mapDomainError(w, err)
		return
	}
    
    h.respond(w, http.StatusOK, map[string]string{"message": "OTP resent"})
}


func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON")
		return
	}

	ip := r.RemoteAddr
	device := r.UserAgent()

	acc, ref, err := h.uc.Login(r.Context(), req.Phone, req.Password, ip, device, req.RememberMe)
	if err != nil {
		h.mapDomainError(w, err)
		return
	}

	h.respond(w, http.StatusOK, TokenResp{AccessToken: acc, RefreshToken: ref})
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
    var req RefreshReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON")
		return
	}
    
    ip := r.RemoteAddr
	device := r.UserAgent()
    
    acc, ref, err := h.uc.RefreshToken(r.Context(), req.RefreshToken, ip, device)
    if err != nil {
		h.mapDomainError(w, err)
		return
	}

	h.respond(w, http.StatusOK, TokenResp{AccessToken: acc, RefreshToken: ref})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value(AuthContextKey("user_id")).(string)
    err := h.uc.Logout(r.Context(), userID)
    if err != nil {
		h.mapDomainError(w, err)
		return
	}
    h.respond(w, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
    var req ForgotReq
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON")
		return
	}
    err := h.uc.ForgotPassword(r.Context(), req.Phone)
    if err != nil {
		h.mapDomainError(w, err)
		return
	}
    h.respond(w, http.StatusOK, map[string]string{"message": "Reset OTP sent"})
}

func (h *AuthHandler) VerifyResetOTP(w http.ResponseWriter, r *http.Request) {
    var req VerifyOTPReq
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}
    if err := h.val.Struct(req); err != nil {
		h.respondError(w, http.StatusUnprocessableEntity, "validation_failed", err.Error())
		return
	}
    
    err := h.uc.VerifyOTP(r.Context(), req.Phone, req.Code, domain.PurposeReset)
    if err != nil {
		h.mapDomainError(w, err)
		return
	}
    
    h.respond(w, http.StatusOK, map[string]string{"message": "OTP verified"})
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
    var req ResetPasswordReq
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}
    if err := h.val.Struct(req); err != nil {
		h.respondError(w, http.StatusUnprocessableEntity, "validation_failed", err.Error())
		return
	}
    
    err := h.uc.ResetPassword(r.Context(), req.Phone, req.Code, req.NewPassword)
    if err != nil {
		h.mapDomainError(w, err)
		return
	}
    
    h.respond(w, http.StatusOK, map[string]string{"message": "Password reset successfully"})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value(AuthContextKey("user_id")).(string)
    user, err := h.uc.GetMe(r.Context(), userID)
    if err != nil {
		h.mapDomainError(w, err)
		return
	}
    
    h.respond(w, http.StatusOK, UserResp{ID: user.ID, FullName: user.FullName, Phone: user.PhoneNumber})
}

func (h *AuthHandler) respond(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(StandardResponse{Data: data})
}

func (h *AuthHandler) respondError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(StandardResponse{Error: &ErrorResp{Code: code, Message: msg}})
}

func (h *AuthHandler) mapDomainError(w http.ResponseWriter, err error) {
	switch err {
	case domain.ErrPhoneExists:
		h.respondError(w, http.StatusConflict, "phone_exists", err.Error())
	case domain.ErrInvalidCredentials:
		h.respondError(w, http.StatusUnauthorized, "invalid_credentials", err.Error())
	case domain.ErrPhoneUnverified:
		h.respondError(w, http.StatusForbidden, "unverified_phone", err.Error())
	case domain.ErrAccountLocked:
		h.respondError(w, http.StatusLocked, "account_locked", err.Error())
	case domain.ErrInvalidOTP:
		h.respondError(w, http.StatusBadRequest, "invalid_otp", err.Error())
    case domain.ErrSessionNotFound:
        h.respondError(w, http.StatusUnauthorized, "invalid_session", err.Error())
    case domain.ErrUnauthorized:
        h.respondError(w, http.StatusUnauthorized, "unauthorized", err.Error())
    case domain.ErrUserNotFound:
        h.respondError(w, http.StatusNotFound, "user_not_found", err.Error())
	default:
		h.respondError(w, http.StatusInternalServerError, "internal_error", "Something went wrong")
	}
}
