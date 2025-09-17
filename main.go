package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"go.uber.org/zap"

	_ "main.go/docs"

	httpSwagger "github.com/swaggo/http-swagger"
)

// BlockAccount represents the account data model
// @Description Block account information with interest calculations
type BlockAccount struct {
	ID           int       `json:"id" example:"1"`
	UserID       int       `json:"user_id" example:"123"`
	Principal    float64   `json:"principal" example:"1000.00"`
	StartDate    time.Time `json:"start_date"`
	EndDate      time.Time `json:"end_date"`
	InterestRate float64   `json:"interest_rate" example:"0.05"`
	Status       string    `json:"status" example:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CreateAccountRequest is the payload for creating accounts
// @Description Request payload for creating a new block account
type CreateAccountRequest struct {
	UserID    int     `json:"user_id" example:"123" binding:"required"`
	Principal float64 `json:"principal" example:"1000.00" binding:"required,gt=0"`
	Period    string  `json:"period" example:"1y" binding:"required"` // "3m", "6m", "1y", "3y"
}

// ErrorResponse represents a standardized error response
// @Description Standard error response format
type ErrorResponse struct {
	Error   string `json:"error" example:"Bad Request"`
	Code    int    `json:"code" example:"400"`
	Message string `json:"message,omitempty" example:"Invalid request body"`
}

// SuccessResponse represents a standardized success response
// @Description Standard success response format
type SuccessResponse struct {
	Success bool        `json:"success" example:"true"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty" example:"Operation completed successfully"`
}

// BlockAccountService interface abstracts business logic
type BlockAccountService interface {
	CreateBlockAccount(ctx context.Context, userID int, principal float64, period string) (*BlockAccount, error)
	GetBlockAccount(ctx context.Context, id int) (*BlockAccount, error)
	GetUserBlockAccounts(ctx context.Context, userID int) ([]*BlockAccount, error)
	DeleteBlockAccount(ctx context.Context, id int) error
}

// service struct is our implementation of BlockAccountService
type service struct {
	db     *sql.DB
	logger *zap.Logger
}

// Context key type for storing service in context
type ctxKey string

const ServiceKey ctxKey = "blockAccountService"

// isValidPeriod validates the period parameter
func isValidPeriod(period string) bool {
	validPeriods := map[string]bool{
		"3m": true, "6m": true, "1y": true, "3y": true,
	}
	return validPeriods[period]
}

// validateCreateRequest validates the create account request
func validateCreateRequest(req *CreateAccountRequest) error {
	if req.UserID <= 0 {
		return fmt.Errorf("user_id must be positive")
	}
	if req.Principal <= 0 {
		return fmt.Errorf("principal must be positive")
	}
	if !isValidPeriod(req.Period) {
		return fmt.Errorf("invalid period: %s. Valid options are: 3m, 6m, 1y, 3y", req.Period)
	}
	return nil
}

// writeError writes a standardized error response
func writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   http.StatusText(statusCode),
		Code:    statusCode,
		Message: message,
	})
}

// writeSuccess writes a standardized success response
func writeSuccess(w http.ResponseWriter, data interface{}, message string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SuccessResponse{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// CreateBlockAccount creates a block account with calculated interest and dates
func (s *service) CreateBlockAccount(ctx context.Context, userID int, principal float64, period string) (*BlockAccount, error) {
	var duration time.Duration
	var interestRate float64

	switch period {
	case "3m":
		duration = time.Hour * 24 * 30 * 3
		interestRate = 0.02
	case "6m":
		duration = time.Hour * 24 * 30 * 6
		interestRate = 0.035
	case "1y":
		duration = time.Hour * 24 * 365
		interestRate = 0.05
	case "3y":
		duration = time.Hour * 24 * 365 * 3
		interestRate = 0.10
	default:
		return nil, fmt.Errorf("invalid period: %s", period)
	}

	startDate := time.Now()
	endDate := startDate.Add(duration)

	var id int
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO block_accounts(user_id, principal, start_date, end_date, interest_rate, status)
         VALUES ($1, $2, $3, $4, $5, 'active') RETURNING id`,
		userID, principal, startDate, endDate, interestRate).Scan(&id)
	if err != nil {
		s.logger.Error("Failed to create block account", zap.Error(err))
		return nil, err
	}

	// Retrieve the full account details
	var account BlockAccount
	err = s.db.QueryRowContext(ctx,
		`SELECT id, user_id, principal, start_date, end_date, interest_rate, status, created_at, updated_at
         FROM block_accounts WHERE id=$1`, id).
		Scan(&account.ID, &account.UserID, &account.Principal, &account.StartDate, &account.EndDate,
			&account.InterestRate, &account.Status, &account.CreatedAt, &account.UpdatedAt)
	if err != nil {
		s.logger.Error("Failed to retrieve created block account", zap.Error(err))
		return nil, err
	}

	return &account, nil
}

// GetBlockAccount retrieves a block account by ID
func (s *service) GetBlockAccount(ctx context.Context, id int) (*BlockAccount, error) {
	var account BlockAccount
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, principal, start_date, end_date, interest_rate, status, created_at, updated_at
         FROM block_accounts WHERE id=$1`, id).
		Scan(&account.ID, &account.UserID, &account.Principal, &account.StartDate, &account.EndDate,
			&account.InterestRate, &account.Status, &account.CreatedAt, &account.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		s.logger.Error("Failed to get block account", zap.Error(err), zap.Int("id", id))
		return nil, err
	}
	return &account, nil
}

// GetUserBlockAccounts retrieves all block accounts for a user
func (s *service) GetUserBlockAccounts(ctx context.Context, userID int) ([]*BlockAccount, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, principal, start_date, end_date, interest_rate, status, created_at, updated_at
         FROM block_accounts WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		s.logger.Error("Failed to get user block accounts", zap.Error(err), zap.Int("userID", userID))
		return nil, err
	}
	defer rows.Close()

	var accounts []*BlockAccount
	for rows.Next() {
		var account BlockAccount
		err := rows.Scan(&account.ID, &account.UserID, &account.Principal, &account.StartDate, &account.EndDate,
			&account.InterestRate, &account.Status, &account.CreatedAt, &account.UpdatedAt)
		if err != nil {
			s.logger.Error("Failed to scan block account", zap.Error(err))
			return nil, err
		}
		accounts = append(accounts, &account)
	}

	if err = rows.Err(); err != nil {
		s.logger.Error("Error iterating block accounts", zap.Error(err))
		return nil, err
	}

	return accounts, nil
}

// DeleteBlockAccount deletes a block account by ID
func (s *service) DeleteBlockAccount(ctx context.Context, id int) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM block_accounts WHERE id=$1`, id)
	if err != nil {
		s.logger.Error("Failed to delete block account", zap.Error(err), zap.Int("id", id))
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		s.logger.Error("Failed to get rows affected", zap.Error(err))
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// Middleware to inject the BlockAccountService into request context
func ServiceMiddleware(svc BlockAccountService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), ServiceKey, svc)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// createBlockAccountHandler godoc
// @Summary Create a new block account
// @Description Creates a new block account with specified user ID, principal, and period
// @Tags block-account
// @Accept json
// @Produce json
// @Param account body CreateAccountRequest true "Create account request"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /block-account [post]
func createBlockAccountHandler(w http.ResponseWriter, r *http.Request) {
	svc, ok := r.Context().Value(ServiceKey).(BlockAccountService)
	if !ok {
		writeError(w, http.StatusInternalServerError, "Service not available")
		return
	}

	var req CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := validateCreateRequest(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	account, err := svc.CreateBlockAccount(ctx, req.UserID, req.Principal, req.Period)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeSuccess(w, account, "Block account created successfully")
}

// getBlockAccountHandler godoc
// @Summary Get block account by ID
// @Description Retrieve a block account by its ID
// @Tags block-account
// @Accept json
// @Produce json
// @Param id path int true "Account ID" Format(int64)
// @Success 200 {object} BlockAccount
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /block-account/{id} [get]
func getBlockAccountHandler(w http.ResponseWriter, r *http.Request) {
	svc, ok := r.Context().Value(ServiceKey).(BlockAccountService)
	if !ok {
		writeError(w, http.StatusInternalServerError, "Service not available")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid block account ID")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	account, err := svc.GetBlockAccount(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if account == nil {
		writeError(w, http.StatusNotFound, "Block account not found")
		return
	}

	writeSuccess(w, account, "Block account retrieved successfully")
}

// getUserBlockAccountsHandler godoc
// @Summary Get all block accounts for a user
// @Description Retrieve all block accounts for a specific user
// @Tags block-account
// @Accept json
// @Produce json
// @Param userID path int true "User ID" Format(int64)
// @Success 200 {array} BlockAccount
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /user/{userID}/block-accounts [get]
func getUserBlockAccountsHandler(w http.ResponseWriter, r *http.Request) {
	svc, ok := r.Context().Value(ServiceKey).(BlockAccountService)
	if !ok {
		writeError(w, http.StatusInternalServerError, "Service not available")
		return
	}

	userIDStr := chi.URLParam(r, "userID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	accounts, err := svc.GetUserBlockAccounts(ctx, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeSuccess(w, accounts, "User block accounts retrieved successfully")
}

// deleteBlockAccountHandler godoc
// @Summary Delete block account by ID
// @Description Deletes a block account by its ID
// @Tags block-account
// @Accept json
// @Produce json
// @Param id path int true "Account ID" Format(int64)
// @Success 204 {string} string "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /block-account/{id} [delete]
func deleteBlockAccountHandler(w http.ResponseWriter, r *http.Request) {
	svc, ok := r.Context().Value(ServiceKey).(BlockAccountService)
	if !ok {
		writeError(w, http.StatusInternalServerError, "Service not available")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid block account ID")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	err = svc.DeleteBlockAccount(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "Block account not found")
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content
}

// healthHandler godoc
// @Summary Health check endpoint
// @Description Check if the service is healthy and database is reachable
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 503 {object} ErrorResponse
// @Router /health [get]
func healthHandler(w http.ResponseWriter, r *http.Request) {
	svc, ok := r.Context().Value(ServiceKey).(BlockAccountService)
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "Service not available")
		return
	}

	// Try to get the service with database connection
	if s, ok := svc.(*service); ok {
		if err := s.db.PingContext(r.Context()); err != nil {
			writeError(w, http.StatusServiceUnavailable, "Database unavailable")
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "timestamp": time.Now().Format(time.RFC3339)})
}

// initDatabase initializes the database schema
func initDatabase(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS block_accounts (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL,
		principal DECIMAL(15,2) NOT NULL,
		start_date TIMESTAMP NOT NULL,
		end_date TIMESTAMP NOT NULL,
		interest_rate DECIMAL(5,4) NOT NULL,
		status VARCHAR(20) DEFAULT 'active',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE INDEX IF NOT EXISTS idx_block_accounts_user_id ON block_accounts(user_id);
	CREATE INDEX IF NOT EXISTS idx_block_accounts_status ON block_accounts(status);
	CREATE INDEX IF NOT EXISTS idx_block_accounts_end_date ON block_accounts(end_date);
	`
	_, err := db.Exec(query)
	return err
}

// @title Block Account API
// @version 1.0
// @description API for managing block accounts with interest calculations
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.example.com/support
// @contact.email support@example.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /
// @schemes http
func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Construct the PostgreSQL DSN from environment variables
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_SSLMODE"),
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test DB connection
	if err := db.Ping(); err != nil {
		logger.Fatal("Cannot reach database", zap.Error(err))
	}

	// Initialize database schema
	if err := initDatabase(db); err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}

	// Create service with logger
	svc := &service{db: db, logger: logger}

	r := chi.NewRouter()

	// Use middlewares for logging and recovery
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Inject service into context via middleware
	r.Use(ServiceMiddleware(svc))

	// Swagger UI route - configure it properly
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"), // The url pointing to API definition
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	))

	// Serve Swagger JSON
	r.Get("/swagger/doc.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		http.ServeFile(w, r, "./docs/swagger.json")
	})

	// Health check route
	r.Get("/health", healthHandler)

	// API routes
	r.Post("/block-account", createBlockAccountHandler)
	r.Get("/block-account/{id}", getBlockAccountHandler)
	r.Get("/user/{userID}/block-accounts", getUserBlockAccountsHandler)
	r.Delete("/block-account/{id}", deleteBlockAccountHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.Info("Server starting",
		zap.String("port", port),
		zap.String("swagger", fmt.Sprintf("http://localhost:%s/swagger/index.html", port)),
	)

	log.Fatal(http.ListenAndServe(":"+port, r))
}
