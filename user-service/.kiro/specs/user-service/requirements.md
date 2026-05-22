# Requirements Document

## Introduction

The User Service is a microservice within a cloud-based delivery express platform responsible for account management. It handles user registration, authentication, profile management, and account lifecycle operations. The service exposes a RESTful API and communicates with other microservices via defined contracts.

## Glossary

- **User Service**: The microservice responsible for all user account management operations.
- **Account**: A registered entity in the system representing a person with a specific role.
- **Customer**: An account role representing a user who places delivery orders.
- **Sender**: An account role representing a user who fulfills and delivers orders (also referred to as courier).
- **Admin**: An account role with elevated privileges to manage other accounts and platform operations.
- **Role**: A label assigned to an account that determines its permissions; valid values are `customer`, `sender`, and `admin`.
- **JWT**: JSON Web Token — a compact, URL-safe token used for stateless authentication.
- **Credentials**: A combination of email and password used to authenticate a user.
- **Access Token**: A short-lived JWT issued upon successful authentication, used to authorize API requests.
- **Refresh Token**: A long-lived token used to obtain a new Access Token without re-authentication.
- **Password Hash**: A one-way cryptographic transformation of a password stored in the database.
- **Profile**: The set of personal information associated with an account (name, phone, address, etc.).
- **Customer Profile**: Profile fields specific to a Customer, including delivery address preferences.
- **Sender Profile**: Profile fields specific to a Sender, including vehicle type and availability status.

---

## Requirements

### Requirement 1

**User Story:** As a new user, I want to register an account with a specific role (customer or sender), so that I can access the delivery platform with the appropriate permissions.

#### Acceptance Criteria

1. WHEN a user submits a registration request with a valid email, password, and a role of either `customer` or `sender`, THE User Service SHALL create a new account and return a success response with the created account ID.
2. WHEN a user submits a registration request with an email that already exists, THE User Service SHALL reject the request and return a conflict error response.
3. WHEN a user submits a registration request with an invalid email format, THE User Service SHALL reject the request and return a validation error response.
4. WHEN a user submits a registration request with a password shorter than 8 characters, THE User Service SHALL reject the request and return a validation error response.
5. WHEN a user submits a registration request with a role value other than `customer` or `sender`, THE User Service SHALL reject the request and return a validation error response.
6. WHEN a new account is created, THE User Service SHALL store the password as a hashed value and SHALL NOT store the plaintext password.

---

### Requirement 2

**User Story:** As a registered user, I want to log in with my credentials, so that I can receive an access token to use the platform.

#### Acceptance Criteria

1. WHEN a user submits valid credentials, THE User Service SHALL return a signed Access Token and a Refresh Token.
2. WHEN a user submits an unrecognized email, THE User Service SHALL return an authentication error response.
3. WHEN a user submits a correct email but incorrect password, THE User Service SHALL return an authentication error response.
4. WHEN a user submits valid credentials, THE User Service SHALL issue an Access Token with an expiry of 15 minutes and a Refresh Token with an expiry of 7 days.

---

### Requirement 3

**User Story:** As an authenticated user, I want to refresh my access token, so that I can maintain my session without re-entering my credentials.

#### Acceptance Criteria

1. WHEN a user submits a valid and unexpired Refresh Token, THE User Service SHALL issue a new Access Token.
2. WHEN a user submits an expired Refresh Token, THE User Service SHALL reject the request and return an authentication error response.
3. WHEN a user submits a malformed or unrecognized Refresh Token, THE User Service SHALL reject the request and return an authentication error response.

---

### Requirement 4

**User Story:** As an authenticated user, I want to view and update my profile, so that I can keep my personal information current.

#### Acceptance Criteria

1. WHEN an authenticated user requests their profile, THE User Service SHALL return the account's profile data excluding the Password Hash.
2. WHEN an authenticated Customer submits a profile update with valid fields, THE User Service SHALL update the Customer Profile and return the updated profile data.
3. WHEN an authenticated Sender submits a profile update with valid fields, THE User Service SHALL update the Sender Profile and return the updated profile data.
4. WHEN an authenticated user submits a profile update with an invalid phone number format, THE User Service SHALL reject the request and return a validation error response.
5. WHILE a profile update is in progress, THE User Service SHALL apply changes atomically so that partial updates are not persisted.

---

### Requirement 5

**User Story:** As an authenticated user, I want to change my password, so that I can maintain the security of my account.

#### Acceptance Criteria

1. WHEN an authenticated user submits a password change request with the correct current password and a valid new password, THE User Service SHALL update the Password Hash and return a success response.
2. WHEN an authenticated user submits a password change request with an incorrect current password, THE User Service SHALL reject the request and return an authentication error response.
3. WHEN an authenticated user submits a new password shorter than 8 characters, THE User Service SHALL reject the request and return a validation error response.

---

### Requirement 6

**User Story:** As an admin, I want to deactivate or reactivate user accounts, so that I can manage platform access and enforce policies.

#### Acceptance Criteria

1. WHEN an admin submits a deactivation request for an active account, THE User Service SHALL set the account status to inactive and return a success response.
2. WHEN an admin submits a reactivation request for an inactive account, THE User Service SHALL set the account status to active and return a success response.
3. WHEN a deactivated user attempts to authenticate, THE User Service SHALL reject the request and return an authorization error response.
4. WHEN a non-admin user attempts to deactivate or reactivate an account, THE User Service SHALL reject the request and return an authorization error response.

---

### Requirement 7

**User Story:** As a Sender, I want to update my availability status, so that the platform knows when I am ready to accept delivery orders.

#### Acceptance Criteria

1. WHEN an authenticated Sender submits an availability update with a valid status of `available` or `unavailable`, THE User Service SHALL update the Sender's availability status and return a success response.
2. WHEN an authenticated Customer attempts to update an availability status, THE User Service SHALL reject the request and return an authorization error response.
3. WHEN a Sender's availability status is updated, THE User Service SHALL persist the change immediately.

---

### Requirement 8

**User Story:** As a developer integrating with the User Service, I want the service to serialize and deserialize account data using JSON, so that other microservices can reliably exchange account information.

#### Acceptance Criteria

1. WHEN the User Service serializes an account object to JSON, THE User Service SHALL produce a JSON representation that includes all required fields and excludes the Password Hash.
2. WHEN the User Service receives a JSON payload, THE User Service SHALL deserialize it into the corresponding internal account structure.
3. WHEN a JSON payload contains unknown fields, THE User Service SHALL ignore unknown fields and process the known fields without error.
4. WHEN a serialized account is deserialized and re-serialized, THE User Service SHALL produce an output equivalent to the original serialized form.
