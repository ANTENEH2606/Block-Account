# Block Account API

    A RESTful API for managing block accounts with interest calculations, built with Go, PostgreSQL, and Chi router. This API allows users to create, retrieve, and delete block accounts with automated interest calculations based on different time periods.

# Features

    Account Management: Create, retrieve, and delete block accounts

    Interest Calculations: Automatic interest rate calculation based on period (3m, 6m, 1y, 3y)

    Swagger Documentation: Comprehensive API documentation with Swagger UI

    Database Integration: PostgreSQL backend with connection pooling

    Structured Logging: Production-ready logging with Zap

    Health Checks: Endpoint for service health monitoring

    Input Validation: Comprehensive request validation

    Error Handling: Standardized error responses

# API Endpoints

    Method	Endpoint	                    Description

    POST	/block-account	                Create a new block account
    GET	    /block-account/{id}	            Get a block account by ID
    GET	    /user/{userID}/block-accounts	Get all block accounts for a user
    DELETE	/block-account/{id}	            Delete a block account by ID
    GET	    /health	                        Health check endpoint
    GET	    /swagger/*	                    Swagger UI documentation

# Interest Rates

    Period	Duration	Interest Rate

    3m	    3 months	2.0%
    6m	    6 months	3.5%
    1y	    1 year	    5.0%
    3y	    3 years	    10.0%

# Prerequisites

Before running this application, ensure you have the following installed:

    Go (version 1.21 or higher)

    PostgreSQL (version 12 or higher)

    Swag (for generating documentation)


# Initialize Go module
    go mod init block-account-api

# Install required packages

    go get github.com/go-chi/chi/v5
    go get github.com/joho/godotenv
    go get github.com/lib/pq
    go get github.com/swaggo/http-swagger
    go get go.uber.org/zap

# Install Swag CLI for documentation generation

    go install github.com/swaggo/swag/cmd/swag@latest

# Database Setup

    Create a PostgreSQL database for the application:

    sql
    CREATE DATABASE block_account_db;

# Environment Configuration

    Create a .env file in the root directory:

    env
    DB_HOST=localhost
    DB_PORT=5432
    DB_USER=your_username
    DB_PASSWORD=password
    DB_NAME=block_account_db
    DB_SSLMODE=disable
    PORT=8080

# Generate Swagger Documentation

    bash

    swag init

    This will create a docs folder with the API documentation.

# Running the Application
   1. Start the Server

        bash

        go run main.go

        The server will start on port 8080 

    2. Access the API

        API Base URL: http://localhost:8080

        Swagger UI: http://localhost:8080/swagger/index.html

        Health Check: http://localhost:8080/health

    API Usage Examples:-

    Create a Block Account

        bash
            curl -X POST "http://localhost:8080/block-account" \
            -H "Content-Type: application/json" \
            -d '{
                "user_id": 123,
                "principal": 1000.00,
                "period": "1y"
            }'

        Response:

            json
            {
            "success": true,
            "data": {
                "id": 1,
                "user_id": 123,
                "principal": 1000,
                "start_date": "2023-10-01T10:00:00Z",
                "end_date": "2024-10-01T10:00:00Z",
                "interest_rate": 0.05,
                "status": "active",
                "created_at": "2023-10-01T10:00:00Z",
                "updated_at": "2023-10-01T10:00:00Z"
            },
            "message": "Block account created successfully"
            }
    Get a Block Account

        bash

                curl -X GET "http://localhost:8080/block-account/1"

    Get User's Block Accounts

        bash

            curl -X GET "http://localhost:8080/user/123/block-accounts"

    Delete a Block Account

        bash

            curl -X DELETE "http://localhost:8080/block-account/1"

Database Schema

    The application automatically creates the following table structure:

    block_accounts Table

    Column	             Type	                               Description

    id	            SERIAL PRIMARY KEY	                    Unique identifier
    user_id	        INTEGER NOT NULL	                    User identifier
    principal	    DECIMAL(15,2) NOT NULL	                Initial investment amount
    start_date	    TIMESTAMP NOT NULL	                    Account  start date
    end_date	    TIMESTAMP NOT NULL	                    Account maturity date
    interest_rate	DECIMAL(5,4) NOT NULL	                Annual interest rate
    status	        VARCHAR(20) DEFAULT 'active'	        Account status
    created_at	    TIMESTAMP DEFAULT CURRENT_TIMESTAMP	    Creation timestamp
    updated_at	    TIMESTAMP DEFAULT CURRENT_TIMESTAMP	    Last update timestamp