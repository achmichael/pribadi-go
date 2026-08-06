package usecase

import (
	"context"
	"crypto/rand"
	"math/big"
	"time"

	"github.com/achmichael/pribadi-go/internal/auth/domain"
	"github.com/achmichael/pribadi-go/internal/auth/infrastructure"
)

type AuthUsecase struct {
	repo      domain.AuthRepository
	otpSender infrastructure.OTPSender
	tokenSvc  *infrastructure.TokenService
}

func NewAuthUsecase(r domain.AuthRepository, o infrastructure.OTPSender, t *infrastructure.TokenService) *AuthUsecase {
	return &AuthUsecase{repo: r, otpSender: o, tokenSvc: t}
}

func (u *AuthUsecase) Register(ctx context.Context, name, phone, password string) error {
	existing, _ := u.repo.GetUserByPhone(phone)
	if existing != nil {
		return domain.ErrPhoneExists
	}

	hash, err := infrastructure.HashPassword(password)
	if err != nil {
		return err
	}

	user := &domain.User{
		FullName:      name,
		PhoneNumber:   phone,
		PasswordHash:  hash,
		PhoneVerified: false,
	}

	if err := u.repo.CreateUser(user); err != nil {
		return err
	}

	return u.generateAndSendOTP(ctx, user.ID, phone, domain.PurposeRegister)
}

func (u *AuthUsecase) VerifyOTP(ctx context.Context, phone, code string, purpose domain.OTPPurpose) error {
	user, err := u.repo.GetUserByPhone(phone)
	if err != nil {
		return domain.ErrUserNotFound
	}

	otp, err := u.repo.GetLatestOTP(user.ID, purpose)
	if err != nil || otp == nil {
		return domain.ErrInvalidOTP
	}

	if otp.AttemptCount >= 3 {
		return domain.ErrMaxOTPAttempts
	}

	otp.AttemptCount++
	_ = u.repo.UpdateOTP(otp)

	if time.Now().UTC().After(otp.ExpiresAt) {
		return domain.ErrInvalidOTP
	}

	if !infrastructure.CheckHash(code, otp.CodeHash) {
		return domain.ErrInvalidOTP
	}

	// Valid. Mark verified if register.
	if purpose == domain.PurposeRegister {
		user.PhoneVerified = true
		_ = u.repo.UpdateUser(user)
	}

	return nil
}

func (u *AuthUsecase) ResendOTP(ctx context.Context, phone string, purpose domain.OTPPurpose) error {
    user, err := u.repo.GetUserByPhone(phone)
	if err != nil {
		return domain.ErrUserNotFound
	}
    
    // Check if there is already an active OTP, prevent spam (e.g., must wait 1 min)
    latestOtp, _ := u.repo.GetLatestOTP(user.ID, purpose)
    if latestOtp != nil {
        if time.Now().UTC().Before(latestOtp.CreatedAt.Add(1 * time.Minute)) {
            return domain.ErrValidation // or ErrTooManyRequests
        }
    }
    
    return u.generateAndSendOTP(ctx, user.ID, phone, purpose)
}

func (u *AuthUsecase) Login(ctx context.Context, phone, password, ip, device string, rememberMe bool) (string, string, error) {
	user, err := u.repo.GetUserByPhone(phone)
	if err != nil {
		return "", "", domain.ErrInvalidCredentials
	}

	if user.LockedUntil != nil && time.Now().UTC().Before(*user.LockedUntil) {
		return "", "", domain.ErrAccountLocked
	}

	if !infrastructure.CheckHash(password, user.PasswordHash) {
		user.FailedAttempts++
		if user.FailedAttempts >= 5 {
			lockTime := time.Now().UTC().Add(15 * time.Minute)
			user.LockedUntil = &lockTime
		}
		_ = u.repo.UpdateUser(user)
		return "", "", domain.ErrInvalidCredentials
	}

	if !user.PhoneVerified {
		return "", "", domain.ErrPhoneUnverified
	}

	// Reset attempts
	user.FailedAttempts = 0
	user.LockedUntil = nil
	_ = u.repo.UpdateUser(user)

	accessToken, _ := u.tokenSvc.GenerateAccessToken(user.ID, 15*time.Minute)
	refreshToken, _ := u.tokenSvc.GenerateRefreshToken()

	rtDuration := 24 * time.Hour
	if rememberMe {
		rtDuration = 7 * 24 * time.Hour
	}

	session := &domain.Session{
		UserID:           user.ID,
		RefreshTokenHash: infrastructure.HashTokenFast(refreshToken),
		DeviceInfo:       device,
		IPAddress:        ip,
		ExpiresAt:        time.Now().UTC().Add(rtDuration),
	}
	_ = u.repo.CreateSession(session)

	return accessToken, refreshToken, nil
}

func (u *AuthUsecase) RefreshToken(ctx context.Context, refToken, ip, device string) (string, string, error) {
    hash := infrastructure.HashTokenFast(refToken)
    session, err := u.repo.GetSessionByHash(hash)
    if err != nil || session == nil {
        return "", "", domain.ErrSessionNotFound
    }
    
    if time.Now().UTC().After(session.ExpiresAt) {
        _ = u.repo.DeleteSession(session.ID)
        return "", "", domain.ErrUnauthorized
    }
    
    // Revoke old session
    _ = u.repo.DeleteSession(session.ID)
    
    // Generate new tokens
    accessToken, _ := u.tokenSvc.GenerateAccessToken(session.UserID, 15*time.Minute)
	newRefreshToken, _ := u.tokenSvc.GenerateRefreshToken()
    
    newSession := &domain.Session{
		UserID:           session.UserID,
		RefreshTokenHash: infrastructure.HashTokenFast(newRefreshToken),
		DeviceInfo:       device,
		IPAddress:        ip,
		ExpiresAt:        time.Now().UTC().Add(7 * 24 * time.Hour), // Reset expiration
	}
    _ = u.repo.CreateSession(newSession)
    
    return accessToken, newRefreshToken, nil
}

func (u *AuthUsecase) Logout(ctx context.Context, userID string) error {
    return u.repo.DeleteAllUserSessions(userID)
}

func (u *AuthUsecase) ForgotPassword(ctx context.Context, phone string) error {
    user, err := u.repo.GetUserByPhone(phone)
	if err != nil {
		return domain.ErrUserNotFound
	}
    return u.generateAndSendOTP(ctx, user.ID, phone, domain.PurposeReset)
}

func (u *AuthUsecase) ResetPassword(ctx context.Context, phone, code, newPassword string) error {
    // 1. Verify OTP first
    if err := u.VerifyOTP(ctx, phone, code, domain.PurposeReset); err != nil {
        return err
    }
    
    user, _ := u.repo.GetUserByPhone(phone)
    
    hash, err := infrastructure.HashPassword(newPassword)
	if err != nil {
		return err
	}
    
    user.PasswordHash = hash
    if err := u.repo.UpdateUser(user); err != nil {
        return err
    }
    
    // Invalidate all existing sessions
    _ = u.repo.DeleteAllUserSessions(user.ID)
    
    return nil
}

func (u *AuthUsecase) GetMe(ctx context.Context, userID string) (*domain.User, error) {
    return u.repo.GetUserByID(userID)
}


func (u *AuthUsecase) generateAndSendOTP(ctx context.Context, userID, phone string, purpose domain.OTPPurpose) error {
	codeStr := generateRandomNumeric(6)
	codeHash, _ := infrastructure.HashPassword(codeStr)

	otp := &domain.OTPCode{
		UserID:       userID,
		CodeHash:     codeHash,
		Purpose:      purpose,
		ExpiresAt:    time.Now().UTC().Add(5 * time.Minute),
		AttemptCount: 0,
	}

	if err := u.repo.SaveOTP(otp); err != nil {
		return err
	}

	return u.otpSender.Send(ctx, phone, codeStr)
}

func generateRandomNumeric(length int) string {
	chars := "0123456789"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		result[i] = chars[num.Int64()]
	}
	return string(result)
}
