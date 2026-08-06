package repository

import (
	"database/sql"
	"errors"

	"github.com/achmichael/pribadi-go/internal/auth/domain"
)

type PostgresAuthRepository struct {
	db *sql.DB
}

func NewPostgresAuthRepository(db *sql.DB) *PostgresAuthRepository {
	return &PostgresAuthRepository{db: db}
}

func (r *PostgresAuthRepository) CreateUser(user *domain.User) error {
	query := `INSERT INTO auth_users (full_name, phone_number, password_hash, phone_verified) 
              VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at`
	return r.db.QueryRow(query, user.FullName, user.PhoneNumber, user.PasswordHash, user.PhoneVerified).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *PostgresAuthRepository) GetUserByPhone(phone string) (*domain.User, error) {
	user := &domain.User{}
	query := `SELECT id, full_name, phone_number, password_hash, phone_verified, failed_attempts, locked_until, created_at, updated_at 
              FROM auth_users WHERE phone_number = $1`
	err := r.db.QueryRow(query, phone).Scan(&user.ID, &user.FullName, &user.PhoneNumber, &user.PasswordHash,
		&user.PhoneVerified, &user.FailedAttempts, &user.LockedUntil, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (r *PostgresAuthRepository) GetUserByID(id string) (*domain.User, error) {
	user := &domain.User{}
	query := `SELECT id, full_name, phone_number, password_hash, phone_verified, failed_attempts, locked_until, created_at, updated_at 
              FROM auth_users WHERE id = $1`
	err := r.db.QueryRow(query, id).Scan(&user.ID, &user.FullName, &user.PhoneNumber, &user.PasswordHash,
		&user.PhoneVerified, &user.FailedAttempts, &user.LockedUntil, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (r *PostgresAuthRepository) UpdateUser(user *domain.User) error {
	query := `UPDATE auth_users SET full_name = $1, password_hash = $2, phone_verified = $3, 
              failed_attempts = $4, locked_until = $5, updated_at = CURRENT_TIMESTAMP WHERE id = $6`
	_, err := r.db.Exec(query, user.FullName, user.PasswordHash, user.PhoneVerified, user.FailedAttempts, user.LockedUntil, user.ID)
	return err
}

func (r *PostgresAuthRepository) SaveOTP(otp *domain.OTPCode) error {
	query := `INSERT INTO auth_otp_codes (user_id, code_hash, purpose, expires_at, attempt_count) 
              VALUES ($1, $2, $3, $4, $5) RETURNING id`
	return r.db.QueryRow(query, otp.UserID, otp.CodeHash, otp.Purpose, otp.ExpiresAt, otp.AttemptCount).Scan(&otp.ID)
}

func (r *PostgresAuthRepository) GetLatestOTP(userID string, purpose domain.OTPPurpose) (*domain.OTPCode, error) {
	otp := &domain.OTPCode{}
	query := `SELECT id, user_id, code_hash, purpose, expires_at, attempt_count, created_at 
              FROM auth_otp_codes WHERE user_id = $1 AND purpose = $2 ORDER BY created_at DESC LIMIT 1`
	err := r.db.QueryRow(query, userID, purpose).Scan(&otp.ID, &otp.UserID, &otp.CodeHash, &otp.Purpose, &otp.ExpiresAt, &otp.AttemptCount, &otp.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrInvalidOTP
		}
		return nil, err
	}
	return otp, nil
}

func (r *PostgresAuthRepository) UpdateOTP(otp *domain.OTPCode) error {
	query := `UPDATE auth_otp_codes SET attempt_count = $1 WHERE id = $2`
	_, err := r.db.Exec(query, otp.AttemptCount, otp.ID)
	return err
}

func (r *PostgresAuthRepository) CreateSession(session *domain.Session) error {
	query := `INSERT INTO auth_sessions (user_id, refresh_token_hash, device_info, ip_address, expires_at) 
              VALUES ($1, $2, $3, $4, $5) RETURNING id`
	return r.db.QueryRow(query, session.UserID, session.RefreshTokenHash, session.DeviceInfo, session.IPAddress, session.ExpiresAt).Scan(&session.ID)
}

func (r *PostgresAuthRepository) GetSessionByHash(hash string) (*domain.Session, error) {
	session := &domain.Session{}
	query := `SELECT id, user_id, refresh_token_hash, device_info, ip_address, expires_at 
              FROM auth_sessions WHERE refresh_token_hash = $1`
	err := r.db.QueryRow(query, hash).Scan(&session.ID, &session.UserID, &session.RefreshTokenHash, &session.DeviceInfo, &session.IPAddress, &session.ExpiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSessionNotFound
		}
		return nil, err
	}
	return session, nil
}

func (r *PostgresAuthRepository) DeleteSession(id string) error {
	query := `DELETE FROM auth_sessions WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *PostgresAuthRepository) DeleteAllUserSessions(userID string) error {
	query := `DELETE FROM auth_sessions WHERE user_id = $1`
	_, err := r.db.Exec(query, userID)
	return err
}
