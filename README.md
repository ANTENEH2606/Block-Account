# Block Account Service
    A RESTful API service for managing fixed-term investment accounts with automated interest calculation and maturity tracking.

# Overview
   This service provides a Go-based backend for creating and managing "block accounts" - fixed-term investment accounts where funds are locked for a specific period with predetermined interest rates. The system supports different investment periods (3 months, 6 months, 1 year, 3 years) with varying interest rates.

# Project Structure
    text
    .
    ├── main.go                 # Main application entry point
    ├── go.mod                 # Go module dependencies
    ├── go.sum                 # Dependency checksums
    ├── .env                   # Environment variables (create this)
    └── README.md             # This file

# Key Components

    ## Data Models

        BlockAccount
        Represents a fixed-term investment account with:

        ID: Unique identifier

        UserID: Owner of the account

        Principal: Initial investment amount

        StartDate: When the investment begins

        EndDate: When the investment matures

        InterestRate: Annual interest rate for the period

        Status: Account status (active/matured)

        CreateAccountRequest
        Request payload for creating new accounts:

        UserID: Account owner identifier

        Principal: Investment amount

        Period: Investment duration ("3m", "6m", "1y", "3y")

    ## Service Layer

        BlockAccountService Interface
        Defines the core business operations:

            CreateBlockAccount(): Creates new investment accounts

            GetBlockAccount(): Retrieves account details

            DeleteBlockAccount(): Removes accounts


    ## HTTP Handlers

        createBlockAccountHandler: POST endpoint for account creation

        getBlockAccountHandler: GET endpoint for account retrieval

        deleteBlockAccountHandler: DELETE endpoint for account removal

    ## Middleware

        ServiceMiddleware: Injects the service instance into request context

        Chi router middleware for logging and recovery

    
# Clone and Initialize
    bash

    git clone <repository-url>
    cd <project-directory>
    go mod download

# Database Setup

    Create a PostgreSQL database and run the following schema:

    sql
        CREATE TABLE block_accounts (
            id SERIAL PRIMARY KEY,
            user_id INTEGER NOT NULL,
            principal DECIMAL(15,2) NOT NULL,
            start_date TIMESTAMP NOT NULL,
            end_date TIMESTAMP NOT NULL,
            interest_rate DECIMAL(5,4) NOT NULL,
            status VARCHAR(20) DEFAULT 'active'
        );

# Environment Configuration

    Create a .env file in the project root:

        DB_HOST=localhost
        DB_PORT=5432
        DB_USER=your_db_user
        DB_PASSWORD=your_db_password
        DB_NAME=your_db_name
        DB_SSLMODE=disable
        PORT=8080

# Running the Application
bash

    # Start the server
        go run main.go

        The server will start on the specified port (default: 8080).

    # API Endpoints

    1. Create Block Account

        POST /block-account

        Request body:

        json
            {
                "user_id": 123,
                "principal": 1000.00,
                "period": "1y"
            }
        Response:

        json
            {
                "message": "Block account created successfully",
                "account": {
                    "id": 1,
                    "user_id": 123,
                    "principal": 1000.00,
                    "start_date": "2023-10-01T10:00:00Z",
                    "end_date": "2024-10-01T10:00:00Z",
                    "interest_rate": 0.05,
                    "status": "active"
            }
        }

    2. Get Block Account

        GET /block-account/{id}

        Response:

            json
            {
                "id": 1,
                "user_id": 123,
                "principal": 1000.00,
                "start_date": "2023-10-01T10:00:00Z",
                "end_date": "2024-10-01T10:00:00Z",
                "interest_rate": 0.05,
                "status": "active"
            }

    3. Delete Block Account

        DELETE /block-account/{id}

        Response: 204 No Content

# Interest Rate Schedule
    Period	Duration	Interest Rate
    3m	3 months	2.0%
    6m	6 months	3.5%
    1y	1 year	5.0%
    3y	3 years	10.0%

# Error Handling
    The API returns appropriate HTTP status codes:

    200: Success

    400: Bad request (invalid input)

    404: Resource not found

    500: Internal server error

# Testing bash

    # Create the account

    curl -X POST -H "Content-Type: application/json" -d '{
    "user_id": 1,
    "principal": 5000.00,
    "period": "1y"
    }' http://localhost:8080/block-account

    # GET the account

    curl http://localhost:8080/block-account/1

    # Delete the account

    curl -X DELETE http://localhost:8080/block-account/1


# Dependencies

    Chi: Lightweight HTTP router

    lib/pq: PostgreSQL driver

    godotenv: Environment variable management