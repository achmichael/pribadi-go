package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/achmichael/pribadi-go/internal/domain"
	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"golang.org/x/crypto/bcrypt"
)

// DashboardService provides business logic for the dashboard
type DashboardService interface {
	// Auth
	Login(ctx context.Context, username, password string) (string, error)
	CreateDefaultUser(ctx context.Context, username, password string) error

	// Config
	GetConfigAll(ctx context.Context) (map[string]domain.AgentConfig, error)
	GetConfigByKey(ctx context.Context, key string) (*domain.AgentConfig, error)
	UpdateConfig(ctx context.Context, key, valueJSON string) error

	// Entities
	ListEntitySchemas(ctx context.Context) ([]domain.CustomEntitySchema, error)
	GetEntitySchema(ctx context.Context, id string) (*domain.CustomEntitySchema, error)
	CreateEntitySchema(ctx context.Context, name, label, description, fieldsJSON string) (*domain.CustomEntitySchema, error)
	UpdateEntitySchema(ctx context.Context, id, name, label, description, fieldsJSON string) error
	DeleteEntitySchema(ctx context.Context, id string) error

	ListEntityRecords(ctx context.Context, schemaID string) ([]domain.CustomEntityRecord, error)
	GetEntityRecord(ctx context.Context, id string) (*domain.CustomEntityRecord, error)
	CreateEntityRecord(ctx context.Context, schemaID, dataJSON string) (*domain.CustomEntityRecord, error)
	UpdateEntityRecord(ctx context.Context, id, dataJSON string) error
	DeleteEntityRecord(ctx context.Context, id string) error

	// Cron Jobs
	ListCronJobs(ctx context.Context) ([]domain.CronJob, error)
	CreateCronJob(ctx context.Context, job domain.CronJob) (*domain.CronJob, error)
	UpdateCronJob(ctx context.Context, job domain.CronJob) error
	DeleteCronJob(ctx context.Context, id string) error

	// Stock Watchlist
	ListStocks(ctx context.Context) ([]domain.StockWatchlist, error)
	AddStock(ctx context.Context, stock domain.StockWatchlist) (*domain.StockWatchlist, error)
	UpdateStock(ctx context.Context, stock domain.StockWatchlist) error
	DeleteStock(ctx context.Context, id string) error
}

type dashboardService struct {
	repo      repository.DashboardRepository
	jwtSecret string
	logger    *zerolog.Logger
}

// NewDashboardService creates a new DashboardService
func NewDashboardService(repo repository.DashboardRepository, jwtSecret string, logger *zerolog.Logger) DashboardService {
	return &dashboardService{
		repo:      repo,
		jwtSecret: jwtSecret,
		logger:    logger,
	}
}

// ─── Auth ─────────────────────────────────────────────────────────────────

func (s *dashboardService) Login(ctx context.Context, username, password string) (string, error) {
	user, err := s.repo.GetDashboardUserByUsername(ctx, username)
	if err != nil {
		return "", fmt.Errorf("user not found or db error: %v", err)
	}
	if user == nil {
		return "", errors.New("invalid credentials")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", err
	}

	_ = s.repo.UpdateUserLastLogin(ctx, user.ID)
	return tokenString, nil
}

func (s *dashboardService) CreateDefaultUser(ctx context.Context, username, password string) error {
	existing, _ := s.repo.GetDashboardUserByUsername(ctx, username)
	if existing != nil {
		return nil // already exists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := domain.DashboardUser{
		ID:           uuid.NewString(),
		Username:     username,
		PasswordHash: string(hash),
	}
	return s.repo.CreateDashboardUser(ctx, user)
}

// ─── Config ───────────────────────────────────────────────────────────────

func (s *dashboardService) GetConfigAll(ctx context.Context) (map[string]domain.AgentConfig, error) {
	return s.repo.GetAgentConfigAll(ctx)
}

func (s *dashboardService) GetConfigByKey(ctx context.Context, key string) (*domain.AgentConfig, error) {
	return s.repo.GetAgentConfigByKey(ctx, key)
}

func (s *dashboardService) UpdateConfig(ctx context.Context, key, valueJSON string) error {
	return s.repo.UpsertAgentConfig(ctx, domain.AgentConfig{
		Key:       key,
		ValueJSON: valueJSON,
	})
}

// ─── Entities ─────────────────────────────────────────────────────────────

func (s *dashboardService) ListEntitySchemas(ctx context.Context) ([]domain.CustomEntitySchema, error) {
	return s.repo.ListEntitySchemas(ctx)
}

func (s *dashboardService) GetEntitySchema(ctx context.Context, id string) (*domain.CustomEntitySchema, error) {
	return s.repo.GetEntitySchema(ctx, id)
}

func (s *dashboardService) CreateEntitySchema(ctx context.Context, name, label, description, fieldsJSON string) (*domain.CustomEntitySchema, error) {
	// Validate JSON Schema
	_, err := jsonschema.CompileString("schema.json", fieldsJSON)
	if err != nil {
		return nil, fmt.Errorf("invalid JSON schema: %w", err)
	}

	schema := domain.CustomEntitySchema{
		ID:          uuid.NewString(),
		Name:        name,
		Label:       label,
		Description: description,
		FieldsJSON:  fieldsJSON,
	}
	err = s.repo.CreateEntitySchema(ctx, schema)
	return &schema, err
}

func (s *dashboardService) UpdateEntitySchema(ctx context.Context, id, name, label, description, fieldsJSON string) error {
	_, err := jsonschema.CompileString("schema.json", fieldsJSON)
	if err != nil {
		return fmt.Errorf("invalid JSON schema: %w", err)
	}

	schema := domain.CustomEntitySchema{
		ID:          id,
		Name:        name,
		Label:       label,
		Description: description,
		FieldsJSON:  fieldsJSON,
	}
	return s.repo.UpdateEntitySchema(ctx, schema)
}

func (s *dashboardService) DeleteEntitySchema(ctx context.Context, id string) error {
	return s.repo.DeleteEntitySchema(ctx, id)
}

func (s *dashboardService) validateEntityData(fieldsJSON, dataJSON string) error {
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("schema.json", strings.NewReader(fieldsJSON)); err != nil {
		return err
	}
	sch, err := compiler.Compile("schema.json")
	if err != nil {
		return err
	}
	var v interface{}
	if err := json.Unmarshal([]byte(dataJSON), &v); err != nil {
		return err
	}
	return sch.Validate(v)
}

func (s *dashboardService) ListEntityRecords(ctx context.Context, schemaID string) ([]domain.CustomEntityRecord, error) {
	return s.repo.ListEntityRecords(ctx, schemaID)
}

func (s *dashboardService) GetEntityRecord(ctx context.Context, id string) (*domain.CustomEntityRecord, error) {
	return s.repo.GetEntityRecord(ctx, id)
}

func (s *dashboardService) CreateEntityRecord(ctx context.Context, schemaID, dataJSON string) (*domain.CustomEntityRecord, error) {
	schema, err := s.repo.GetEntitySchema(ctx, schemaID)
	if err != nil || schema == nil {
		return nil, fmt.Errorf("schema not found: %v", err)
	}

	if err := s.validateEntityData(schema.FieldsJSON, dataJSON); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	record := domain.CustomEntityRecord{
		ID:       uuid.NewString(),
		SchemaID: schemaID,
		DataJSON: dataJSON,
	}
	err = s.repo.CreateEntityRecord(ctx, record)
	return &record, err
}

func (s *dashboardService) UpdateEntityRecord(ctx context.Context, id, dataJSON string) error {
	record, err := s.repo.GetEntityRecord(ctx, id)
	if err != nil || record == nil {
		return fmt.Errorf("record not found: %v", err)
	}

	schema, err := s.repo.GetEntitySchema(ctx, record.SchemaID)
	if err != nil || schema == nil {
		return fmt.Errorf("schema not found: %v", err)
	}

	if err := s.validateEntityData(schema.FieldsJSON, dataJSON); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	record.DataJSON = dataJSON
	return s.repo.UpdateEntityRecord(ctx, *record)
}

func (s *dashboardService) DeleteEntityRecord(ctx context.Context, id string) error {
	return s.repo.DeleteEntityRecord(ctx, id)
}

// ─── Cron Jobs ────────────────────────────────────────────────────────────

func (s *dashboardService) ListCronJobs(ctx context.Context) ([]domain.CronJob, error) {
	return s.repo.ListCronJobs(ctx)
}

func (s *dashboardService) CreateCronJob(ctx context.Context, job domain.CronJob) (*domain.CronJob, error) {
	job.ID = uuid.NewString()
	err := s.repo.CreateCronJob(ctx, job)
	return &job, err
}

func (s *dashboardService) UpdateCronJob(ctx context.Context, job domain.CronJob) error {
	return s.repo.UpdateCronJob(ctx, job)
}

func (s *dashboardService) DeleteCronJob(ctx context.Context, id string) error {
	return s.repo.DeleteCronJob(ctx, id)
}

// ─── Stock Watchlist ──────────────────────────────────────────────────────

func (s *dashboardService) ListStocks(ctx context.Context) ([]domain.StockWatchlist, error) {
	return s.repo.ListStockWatchlist(ctx)
}

func (s *dashboardService) AddStock(ctx context.Context, stock domain.StockWatchlist) (*domain.StockWatchlist, error) {
	stock.ID = uuid.NewString()
	err := s.repo.CreateStockWatchlist(ctx, stock)
	return &stock, err
}

func (s *dashboardService) UpdateStock(ctx context.Context, stock domain.StockWatchlist) error {
	return s.repo.UpdateStockWatchlist(ctx, stock)
}

func (s *dashboardService) DeleteStock(ctx context.Context, id string) error {
	return s.repo.DeleteStockWatchlist(ctx, id)
}
