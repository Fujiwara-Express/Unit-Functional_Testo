# Implementation Plan — User Service

- [x] 1. Initialize Go module and project structure





  - Run `go mod init` and create the directory layout: `cmd/`, `internal/domain/`, `internal/repository/`, `internal/service/auth/`, `internal/service/user/`, `internal/token/`, `internal/middleware/`, `internal/transport/http/`, `internal/config/`, `migrations/`
  - Add dependencies: `github.com/gin-gonic/gin`, `github.com/golang-jwt/jwt/v5`, `golang.org/x/crypto`, `github.com/jmoiron/sqlx`, `github.com/lib/pq`, `pgregory.net/rapid`
  - Create `cmd/main.go` as the entry point that wires all components together
  - _Requirements: 1.1, 2.1_

- [x] 2. Define domain types and interfaces





  - [x] 2.1 Create domain models and constants in `internal/domain/`


    - Define `User`, `RefreshToken`, `Role` types and `RoleCustomer`, `RoleCourier`, `RoleAdmin` constants
    - Define `AvailabilityStatus` constants
    - _Requirements: 1.1, 1.5_
  - [x] 2.2 Define repository interfaces in `internal/repository/interfaces.go`


    - Define `UserRepository` and `RefreshTokenRepository` interfaces as specified in the design
    - _Requirements: 1.1, 2.1_
  - [x] 2.3 Define service interfaces in `internal/service/`


    - Define `AuthService` and `UserService` interfaces as specified in the design
    - _Requirements: 1.1, 2.1, 4.1_

- [x] 3. Implement input validation utilities





  - [x] 3.1 Create `internal/domain/validation.go` with validators for email format, password length (>= 8), phone format, and role values


    - _Requirements: 1.3, 1.4, 1.5, 4.4_
  - [x] 3.2 Write property tests for input validation


    - **Property 2: Invalid email is always rejected** — generate non-email strings, assert rejection
    - **Property 3: Short password is always rejected** — generate passwords of length 0–7, assert rejection
    - **Property 4: Invalid role is always rejected** — generate arbitrary non-role strings, assert rejection
    - **Property 13: Invalid phone number is always rejected** — generate non-phone strings, assert rejection
    - _Validates: Requirements 1.3, 1.4, 1.5, 4.4_

- [x] 4. Implement JWT Token Manager






  - [x] 4.1 Create `internal/token/jwt.go` implementing `TokenManager`

    - `GenerateAccessToken` signs a JWT with `user_id`, `role`, and `exp` set to now + 3600s
    - `GenerateRefreshToken` signs a JWT with `user_id` and `exp` set to now + 7 days
    - `ValidateAccessToken` and `ValidateRefreshToken` parse and verify signatures and expiry
    - _Requirements: 2.1, 2.4, 3.1_

  - [x] 4.2 Write property tests for Token Manager

    - **Property 8: Token expiry is within expected bounds** — generate valid user IDs and roles, call GenerateAccessToken, decode JWT, assert exp is within 60s of now+3600
    - **Property 9: Valid refresh token always yields a new access token** — generate token, call ValidateRefreshToken, assert claims are non-nil
    - **Property 10: Malformed refresh token is always rejected** — generate random non-JWT strings, assert ValidateRefreshToken returns error
    - _Validates: Requirements 2.4, 3.1, 3.3_

- [x] 5. Implement database migrations





  - Create `migrations/001_create_users.sql` and `migrations/002_create_refresh_tokens.sql` with the schema from the design document
  - _Requirements: 1.1, 2.1_

- [x] 6. Implement User Repository





  - [x] 6.1 Create `internal/repository/user_repo.go` implementing `UserRepository`


    - Implement `Create`, `FindByEmail`, `FindByID`, `Update`, `SetStatus` using `sqlx`
    - _Requirements: 1.1, 1.2, 4.1, 6.1_
  - [x] 6.2 Create `internal/repository/token_repo.go` implementing `RefreshTokenRepository`


    - Implement `Save`, `FindByTokenHash`, `Revoke`, `DeleteExpired`
    - _Requirements: 2.1, 3.1, (Logout)_

- [x] 7. Implement Auth Service





  - [x] 7.1 Create `internal/service/auth/auth_service.go` implementing `AuthService`


    - `Register`: validate inputs, check email uniqueness, bcrypt hash password, persist user, return `user_id`
    - `Login`: find user by email, verify password hash, check `is_active`, generate token pair, persist refresh token hash
    - `RefreshToken`: validate refresh token JWT, look up token hash in DB, generate new access token
    - `Logout`: validate refresh token, revoke token hash in DB
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 2.1, 2.2, 2.3, 2.4, 3.1, 3.2, 3.3, 6.3_

  - [x] 7.2 Write property tests for Auth Service

    - **Property 1: Valid registration always creates a user** — generate valid inputs, call Register, assert non-empty user_id
    - **Property 5: Password is never stored in plaintext** — register user, fetch from repo, assert hash != plaintext
    - **Property 6: Valid login always returns a token pair** — register then login, assert both tokens non-empty
    - **Property 7: Wrong password always fails login** — register user, login with different password, assert auth error
    - **Property 17: Deactivated user cannot authenticate** — register, deactivate, login, assert auth error
    - **Property 19: Logout revokes the refresh token** — login, logout, refresh with same token, assert auth error
    - _Validates: Requirements 1.1, 1.6, 2.1, 2.3, 6.3, (Logout)_

- [x] 8. Checkpoint — Ensure all tests pass, ask the user if questions arise.






- [x] 9. Implement User Service





  - [x] 9.1 Create `internal/service/user/user_service.go` implementing `UserService`


    - `GetProfile`: fetch user by ID, return profile struct without `password_hash`
    - `UpdateProfile`: validate name and phone, update user record, return status `UPDATED`
    - `SetAccountStatus`: verify caller is admin, set `is_active` flag
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 6.1, 6.2, 6.4_
  - [x] 9.2 Write property tests for User Service


    - **Property 11: Profile response never contains password hash** — generate users, call GetProfile, assert no password_hash in JSON output
    - **Property 12: Valid profile update is reflected in the response** — generate valid name/phone, update, get profile, assert values match
    - **Property 14: Correct current password allows password change** — change password, assert new hash differs from old
    - **Property 15: Wrong current password blocks password change** — submit wrong current password, assert auth error
    - **Property 16: Account deactivation/reactivation round-trip** — deactivate then reactivate, assert is_active returns to true
    - **Property 18: Non-admin cannot change account status** — call SetAccountStatus as CUSTOMER/COURIER, assert 403
    - _Validates: Requirements 4.1, 4.2, 4.3, 5.1, 5.2, 6.1, 6.2, 6.4_

- [x] 10. Implement JWT Auth Middleware





  - Create `internal/middleware/auth.go`
  - Extract and validate `Authorization: Bearer <token>` header
  - Inject `user_id` and `role` into Gin context
  - Return 401 for missing or invalid tokens
  - _Requirements: 2.1, 6.4_

- [x] 11. Implement HTTP Handlers and Router






  - [x] 11.1 Create handler files in `internal/transport/http/`

    - `auth_handler.go`: Register, Login, RefreshToken, Logout handlers
    - `user_handler.go`: GetProfile, UpdateProfile handlers
    - `admin_handler.go`: SetAccountStatus handler (admin role check)
    - Each handler parses JSON request, calls the appropriate service method, returns JSON response
    - _Requirements: 1.1, 2.1, 3.1, 4.1, 4.2, 6.1, 6.4_
  - [x] 11.2 Create `internal/transport/http/router.go`


    - Register all routes as defined in the design document
    - Apply auth middleware to protected routes
    - _Requirements: 1.1, 2.1, 4.1, 6.4_
  - [x] 11.3 Write property tests for JSON serialization


    - **Property 20: JSON serialization round-trip** — generate User objects, marshal to JSON, unmarshal, re-marshal, assert equivalence
    - **Property 21: Unknown JSON fields are ignored on deserialization** — generate valid JSON with extra fields, unmarshal, assert known fields correct
    - _Validates: Requirements 8.3, 8.4_

- [x] 12. Wire everything together in `cmd/main.go`





  - Load config (DB DSN, JWT secret, port) from environment variables
  - Initialize DB connection, run migrations
  - Instantiate repositories, token manager, services, and router
  - Start the HTTP server
  - _Requirements: 1.1, 2.1_

- [x] 13. Final Checkpoint — Ensure all tests pass, ask the user if questions arise.





