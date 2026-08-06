package domain

import (
	"errors"
	"time"
)

// Errors
var (
	ErrUserNotFound         = errors.New("user not found")
	ErrPhoneExists          = errors.New("phone number already registered")
	ErrPhoneUnverified      = errors.New("phone number not verified")
	ErrInvalidCredentials   = errors.New("invalid phone number or password")
	ErrAccountLocked        = errors.New("account locked due to failed attempts")
	ErrInvalidOTP           = errors.New("invalid or expired OTP")
	ErrMaxOTPAttempts       = errors.New("max OTP attempts reached")
	ErrSessionNotFound      = errors.New("session not found")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrValidation           = errors.New("validation failed")
)

type User struct {
	ID             string
	FullName       string
	PhoneNumber    string
	PasswordHash   string
	PhoneVerified  bool
	FailedAttempts int
	LockedUntil    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type OTPPurpose string

const (
	PurposeRegister OTPPurpose = "REGISTER"
	PurposeReset    OTPPurpose = "RESET_PASSWORD"
)

type OTPCode struct {
	ID           string
	UserID       string
	CodeHash     string
	Purpose      OTPPurpose
	ExpiresAt    time.Time
	AttemptCount int
    CreatedAt    time.Time
}

type Session struct {
	ID               string
	UserID           string
	RefreshTokenHash string
	DeviceInfo       string
	IPAddress        string
	ExpiresAt        time.Time
}

type AuthRepository interface {
	CreateUser(user *User) error
	GetUserByPhone(phone string) (*User, error)
	GetUserByID(id string) (*User, error)
	UpdateUser(user *User) error
	
	SaveOTP(otp *OTPCode) error
	GetLatestOTP(userID string, purpose OTPPurpose) (*OTPCode, error)
	UpdateOTP(otp *OTPCode) error
	
	CreateSession(session *Session) error
	GetSessionByHash(hash string) (*Session, error)
	DeleteSession(id string) error
	DeleteAllUserSessions(userID string) error
}
