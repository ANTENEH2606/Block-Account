package main

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"
    "strconv"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    _ "github.com/lib/pq"
    "github.com/joho/godotenv"
)

// BlockAccount represents the account data model
type BlockAccount struct {
    ID           int       `json:"id"`
    UserID       int       `json:"user_id"`
    Principal    float64   `json:"principal"`
    StartDate    time.Time `json:"start_date"`
    EndDate      time.Time `json:"end_date"`
    InterestRate float64   `json:"interest_rate"`
    Status       string    `json:"status"`
}

// CreateAccountRequest is the payload for creating accounts
type CreateAccountRequest struct {
    UserID    int     `json:"user_id"`
    Principal float64 `json:"principal"`
    Period    string  `json:"period"` // "3m", "6m", "1y", "3y"
}

// BlockAccountService interface abstracts business logic
type BlockAccountService interface {
    CreateBlockAccount(ctx context.Context, userID int, principal float64, period string) (*BlockAccount, error)
    GetBlockAccount(ctx context.Context, id int) (*BlockAccount, error)
    DeleteBlockAccount(ctx context.Context, id int) error
}

// service struct is our implementation of BlockAccountService
type service struct {
    db *sql.DB
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
        return nil, err
    }

    return &BlockAccount{
        ID:           id,
        UserID:       userID,
        Principal:    principal,
        StartDate:    startDate,
        EndDate:      endDate,
        InterestRate: interestRate,
        Status:       "active",
    }, nil
}

func (s *service) GetBlockAccount(ctx context.Context, id int) (*BlockAccount, error) {
    var account BlockAccount
    err := s.db.QueryRowContext(ctx,
        `SELECT id, user_id, principal, start_date, end_date, interest_rate, status
         FROM block_accounts WHERE id=$1`, id).
        Scan(&account.ID, &account.UserID, &account.Principal, &account.StartDate, &account.EndDate,
            &account.InterestRate, &account.Status)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, nil
        }
        return nil, err
    }
    return &account, nil
}

func (s *service) DeleteBlockAccount(ctx context.Context, id int) error {
    result, err := s.db.ExecContext(ctx, `DELETE FROM block_accounts WHERE id=$1`, id)
    if err != nil {
        return err
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return err
    }
    if rowsAffected == 0 {
        return sql.ErrNoRows
    }
    return nil
}


// Context key type for storing service in context
type ctxKey string

const ServiceKey ctxKey = "blockAccountService"

// Middleware to inject the BlockAccountService into request context
func ServiceMiddleware(svc BlockAccountService) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ctx := context.WithValue(r.Context(), ServiceKey, svc)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// Handler for creating block account
func createBlockAccountHandler(w http.ResponseWriter, r *http.Request) {
    svc, ok := r.Context().Value(ServiceKey).(BlockAccountService)
    if !ok {
        http.Error(w, "Service not available", http.StatusInternalServerError)
        return
    }

    var req CreateAccountRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    account, err := svc.CreateBlockAccount(r.Context(), req.UserID, req.Principal, req.Period)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": "Block account created successfully",
        "account": account,
    })
}

func getBlockAccountHandler(w http.ResponseWriter, r *http.Request) {
    svc, ok := r.Context().Value(ServiceKey).(BlockAccountService)
    if !ok {
        http.Error(w, "Service not available", http.StatusInternalServerError)
        return
    }

    idStr := chi.URLParam(r, "id")
    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Invalid block account ID", http.StatusBadRequest)
        return
    }

    account, err := svc.GetBlockAccount(r.Context(), id)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    if account == nil {
        http.Error(w, "Block account not found", http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(account)
}

func deleteBlockAccountHandler(w http.ResponseWriter, r *http.Request) {
    svc, ok := r.Context().Value(ServiceKey).(BlockAccountService)
    if !ok {
        http.Error(w, "Service not available", http.StatusInternalServerError)
        return
    }

    idStr := chi.URLParam(r, "id")
    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Invalid block account ID", http.StatusBadRequest)
        return
    }

    err = svc.DeleteBlockAccount(r.Context(), id)
    if err != nil {
        if err == sql.ErrNoRows {
            http.Error(w, "Block account not found", http.StatusNotFound)
        } else {
            http.Error(w, err.Error(), http.StatusInternalServerError)
        }
        return
    }

    w.WriteHeader(http.StatusNoContent) // 204 No Content
}


func main() {
    // Load .env file
    if err := godotenv.Load(); err != nil {
        log.Println("No .env file found, relying on environment variables")
    }

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
        log.Fatalf("Failed to connect to database: %v", err)
    }
    defer db.Close()

    // Test DB connection
    if err := db.Ping(); err != nil {
        log.Fatalf("Cannot reach database: %v", err)
    }

    svc := &service{db: db}

    r := chi.NewRouter()

    // Use middlewares for logging and recovery
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)

    // Inject service into context via middleware
    r.Use(ServiceMiddleware(svc))

    // Route for creating block account
    r.Post("/block-account", createBlockAccountHandler)
    r.Get("/block-account/{id}", getBlockAccountHandler)
    r.Delete("/block-account/{id}", deleteBlockAccountHandler)


    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    fmt.Printf("Server listening on :%s\n", port)
    log.Fatal(http.ListenAndServe(":"+port, r))
}
