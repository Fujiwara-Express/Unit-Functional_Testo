package repository_test

import (
"context"
"database/sql"
"database/sql/driver"
"errors"
"testing"
"time"

sqlmock "github.com/DATA-DOG/go-sqlmock"
"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
"pgregory.net/rapid"

"order-service/repository"
"order-service/types"
)

// suppress unused import
var _ = sql.ErrNoRows

// columns is the ordered list of columns used in SELECT queries.
var columns = []string{
"order_id", "tracking_number", "sender_user_id", "sender_name", "sender_address",
"sender_phone", "sender_city_code", "receiver_name", "receiver_address", "receiver_phone",
"receiver_city_code", "weight", "length", "width", "height", "service_type", "price",
"is_cod", "cod_amount", "insurance", "item_description", "status", "created_at", "updated_at",
}

// sampleOrder returns a fully populated Order for use in tests.
func sampleOrder() *types.Order {
now := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
return &types.Order{
OrderID:          "order-001",
TrackingNumber:   "TRK123456",
SenderUserID:     "user-001",
SenderName:       "Alice",
SenderAddress:    "123 Main St",
SenderPhone:      "08111111111",
SenderCityCode:   "JKT",
ReceiverName:     "Bob",
ReceiverAddress:  "456 Oak Ave",
ReceiverPhone:    "08222222222",
ReceiverCityCode: "SBY",
Weight:           2.5,
Length:           30.0,
Width:            20.0,
Height:           15.0,
ServiceType:      types.ServiceTypeREG,
Price:            22500.0,
IsCOD:            false,
CODAmount:        0.0,
Insurance:        true,
ItemDescription:  "Electronics",
Status:           types.OrderStatusAwaitingPickup,
CreatedAt:        now,
UpdatedAt:        now,
}
}

// addOrderRow adds a row to sqlmock rows from an Order.
func addOrderRow(rows *sqlmock.Rows, o *types.Order) *sqlmock.Rows {
return rows.AddRow(
o.OrderID, o.TrackingNumber, o.SenderUserID, o.SenderName, o.SenderAddress,
o.SenderPhone, o.SenderCityCode, o.ReceiverName, o.ReceiverAddress, o.ReceiverPhone,
o.ReceiverCityCode, o.Weight, o.Length, o.Width, o.Height,
string(o.ServiceType), o.Price, o.IsCOD, o.CODAmount, o.Insurance,
o.ItemDescription, string(o.Status), o.CreatedAt, o.UpdatedAt,
)
}

// ─── Task 16.1: SaveOrder tests ───────────────────────────────────────────────

func TestSaveOrder_ValidOrder(t *testing.T) {
db, mock, err := sqlmock.New()
require.NoError(t, err)
defer db.Close()

order := sampleOrder()

mock.ExpectExec("INSERT INTO orders").
WithArgs(
order.OrderID, order.TrackingNumber, order.SenderUserID, order.SenderName,
order.SenderAddress, order.SenderPhone, order.SenderCityCode,
order.ReceiverName, order.ReceiverAddress, order.ReceiverPhone, order.ReceiverCityCode,
order.Weight, order.Length, order.Width, order.Height,
string(order.ServiceType), order.Price, order.IsCOD, order.CODAmount, order.Insurance,
order.ItemDescription, string(order.Status), order.CreatedAt, order.UpdatedAt,
).
WillReturnResult(sqlmock.NewResult(1, 1))

repo := repository.NewOrderRepository(db)
err = repo.SaveOrder(context.Background(), order)
require.NoError(t, err)
require.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveOrder_DBError(t *testing.T) {
db, mock, err := sqlmock.New()
require.NoError(t, err)
defer db.Close()

order := sampleOrder()
dbErr := errors.New("connection refused")

mock.ExpectExec("INSERT INTO orders").
WithArgs(
order.OrderID, order.TrackingNumber, order.SenderUserID, order.SenderName,
order.SenderAddress, order.SenderPhone, order.SenderCityCode,
order.ReceiverName, order.ReceiverAddress, order.ReceiverPhone, order.ReceiverCityCode,
order.Weight, order.Length, order.Width, order.Height,
string(order.ServiceType), order.Price, order.IsCOD, order.CODAmount, order.Insurance,
order.ItemDescription, string(order.Status), order.CreatedAt, order.UpdatedAt,
).
WillReturnError(dbErr)

repo := repository.NewOrderRepository(db)
err = repo.SaveOrder(context.Background(), order)
require.Error(t, err)
assert.Equal(t, dbErr, err)
require.NoError(t, mock.ExpectationsWereMet())
}

// ─── Task 16.2: FindOrderByID tests ──────────────────────────────────────────

func TestFindOrderByID_Found(t *testing.T) {
db, mock, err := sqlmock.New()
require.NoError(t, err)
defer db.Close()

order := sampleOrder()
rows := sqlmock.NewRows(columns)
addOrderRow(rows, order)

mock.ExpectQuery("SELECT").
WithArgs(order.OrderID).
WillReturnRows(rows)

repo := repository.NewOrderRepository(db)
got, err := repo.FindOrderByID(context.Background(), order.OrderID)
require.NoError(t, err)
require.NotNil(t, got)

assert.Equal(t, order.OrderID, got.OrderID)
assert.Equal(t, order.TrackingNumber, got.TrackingNumber)
assert.Equal(t, order.SenderUserID, got.SenderUserID)
assert.Equal(t, order.SenderName, got.SenderName)
assert.Equal(t, order.SenderAddress, got.SenderAddress)
assert.Equal(t, order.SenderPhone, got.SenderPhone)
assert.Equal(t, order.SenderCityCode, got.SenderCityCode)
assert.Equal(t, order.ReceiverName, got.ReceiverName)
assert.Equal(t, order.ReceiverAddress, got.ReceiverAddress)
assert.Equal(t, order.ReceiverPhone, got.ReceiverPhone)
assert.Equal(t, order.ReceiverCityCode, got.ReceiverCityCode)
assert.Equal(t, order.Weight, got.Weight)
assert.Equal(t, order.Length, got.Length)
assert.Equal(t, order.Width, got.Width)
assert.Equal(t, order.Height, got.Height)
assert.Equal(t, order.ServiceType, got.ServiceType)
assert.Equal(t, order.Price, got.Price)
assert.Equal(t, order.IsCOD, got.IsCOD)
assert.Equal(t, order.CODAmount, got.CODAmount)
assert.Equal(t, order.Insurance, got.Insurance)
assert.Equal(t, order.ItemDescription, got.ItemDescription)
assert.Equal(t, order.Status, got.Status)
assert.Equal(t, order.CreatedAt, got.CreatedAt)
assert.Equal(t, order.UpdatedAt, got.UpdatedAt)

require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindOrderByID_NotFound(t *testing.T) {
db, mock, err := sqlmock.New()
require.NoError(t, err)
defer db.Close()

rows := sqlmock.NewRows(columns)
mock.ExpectQuery("SELECT").
WithArgs("nonexistent-id").
WillReturnRows(rows)

repo := repository.NewOrderRepository(db)
got, err := repo.FindOrderByID(context.Background(), "nonexistent-id")
require.Error(t, err)
assert.Nil(t, got)
assert.ErrorIs(t, err, types.ErrNotFound)
require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindOrderByID_DBError(t *testing.T) {
db, mock, err := sqlmock.New()
require.NoError(t, err)
defer db.Close()

dbErr := errors.New("query timeout")
mock.ExpectQuery("SELECT").
WithArgs("order-001").
WillReturnError(dbErr)

repo := repository.NewOrderRepository(db)
got, err := repo.FindOrderByID(context.Background(), "order-001")
require.Error(t, err)
assert.Nil(t, got)
assert.Equal(t, dbErr, err)
require.NoError(t, mock.ExpectationsWereMet())
}

// ─── Task 16.3: FindOrders tests ─────────────────────────────────────────────

func TestFindOrders_WithUserID(t *testing.T) {
db, mock, err := sqlmock.New()
require.NoError(t, err)
defer db.Close()

order := sampleOrder()
rows := sqlmock.NewRows(columns)
addOrderRow(rows, order)

mock.ExpectQuery("WHERE sender_user_id").
WithArgs("user-001", 10, 0).
WillReturnRows(rows)

repo := repository.NewOrderRepository(db)
params := types.ListOrdersParams{UserID: "user-001", Page: 1, Limit: 10}
got, err := repo.FindOrders(context.Background(), params)
require.NoError(t, err)
assert.Len(t, got, 1)
require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindOrders_WithStatus(t *testing.T) {
db, mock, err := sqlmock.New()
require.NoError(t, err)
defer db.Close()

order := sampleOrder()
rows := sqlmock.NewRows(columns)
addOrderRow(rows, order)

mock.ExpectQuery("WHERE status").
WithArgs(string(types.OrderStatusAwaitingPickup), 10, 0).
WillReturnRows(rows)

repo := repository.NewOrderRepository(db)
params := types.ListOrdersParams{Status: types.OrderStatusAwaitingPickup, Page: 1, Limit: 10}
got, err := repo.FindOrders(context.Background(), params)
require.NoError(t, err)
assert.Len(t, got, 1)
require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindOrders_WithUserIDAndStatus(t *testing.T) {
db, mock, err := sqlmock.New()
require.NoError(t, err)
defer db.Close()

order := sampleOrder()
rows := sqlmock.NewRows(columns)
addOrderRow(rows, order)

mock.ExpectQuery("WHERE sender_user_id").
WithArgs("user-001", string(types.OrderStatusCreated), 10, 0).
WillReturnRows(rows)

repo := repository.NewOrderRepository(db)
params := types.ListOrdersParams{
UserID: "user-001",
Status: types.OrderStatusCreated,
Page:   1,
Limit:  10,
}
got, err := repo.FindOrders(context.Background(), params)
require.NoError(t, err)
assert.Len(t, got, 1)
require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindOrders_NoFilters(t *testing.T) {
db, mock, err := sqlmock.New()
require.NoError(t, err)
defer db.Close()

order := sampleOrder()
rows := sqlmock.NewRows(columns)
addOrderRow(rows, order)

// No WHERE clause — just ORDER BY LIMIT OFFSET
mock.ExpectQuery("ORDER BY created_at DESC").
WithArgs(10, 0).
WillReturnRows(rows)

repo := repository.NewOrderRepository(db)
params := types.ListOrdersParams{Page: 1, Limit: 10}
got, err := repo.FindOrders(context.Background(), params)
require.NoError(t, err)
assert.Len(t, got, 1)
require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindOrders_DBError(t *testing.T) {
db, mock, err := sqlmock.New()
require.NoError(t, err)
defer db.Close()

dbErr := errors.New("db unavailable")
mock.ExpectQuery("SELECT").
WillReturnError(dbErr)

repo := repository.NewOrderRepository(db)
params := types.ListOrdersParams{Page: 1, Limit: 10}
got, err := repo.FindOrders(context.Background(), params)
require.Error(t, err)
assert.Nil(t, got)
assert.Equal(t, dbErr, err)
require.NoError(t, mock.ExpectationsWereMet())
}

// ─── Task 16.4: UpdateOrder tests ────────────────────────────────────────────

func TestUpdateOrder_ValidOrder(t *testing.T) {
db, mock, err := sqlmock.New()
require.NoError(t, err)
defer db.Close()

order := sampleOrder()

mock.ExpectExec("UPDATE orders").
WithArgs(
order.TrackingNumber, order.SenderUserID, order.SenderName, order.SenderAddress,
order.SenderPhone, order.SenderCityCode,
order.ReceiverName, order.ReceiverAddress, order.ReceiverPhone, order.ReceiverCityCode,
order.Weight, order.Length, order.Width, order.Height,
string(order.ServiceType), order.Price, order.IsCOD, order.CODAmount, order.Insurance,
order.ItemDescription, string(order.Status), order.UpdatedAt,
order.OrderID,
).
WillReturnResult(sqlmock.NewResult(1, 1))

repo := repository.NewOrderRepository(db)
err = repo.UpdateOrder(context.Background(), order)
require.NoError(t, err)
require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateOrder_DBError(t *testing.T) {
db, mock, err := sqlmock.New()
require.NoError(t, err)
defer db.Close()

order := sampleOrder()
dbErr := errors.New("update failed")

mock.ExpectExec("UPDATE orders").
WithArgs(
order.TrackingNumber, order.SenderUserID, order.SenderName, order.SenderAddress,
order.SenderPhone, order.SenderCityCode,
order.ReceiverName, order.ReceiverAddress, order.ReceiverPhone, order.ReceiverCityCode,
order.Weight, order.Length, order.Width, order.Height,
string(order.ServiceType), order.Price, order.IsCOD, order.CODAmount, order.Insurance,
order.ItemDescription, string(order.Status), order.UpdatedAt,
order.OrderID,
).
WillReturnError(dbErr)

repo := repository.NewOrderRepository(db)
err = repo.UpdateOrder(context.Background(), order)
require.Error(t, err)
assert.Equal(t, dbErr, err)
require.NoError(t, mock.ExpectationsWereMet())
}

// ─── Task 16.5: Property 9 — Repository Round-Trip Fidelity ──────────────────
// Validates: Requirements 11.1, 11.2

func TestProperty9_RepositoryRoundTripFidelity(t *testing.T) {
serviceTypes := []types.ServiceType{
types.ServiceTypeREG, types.ServiceTypeYES, types.ServiceTypeOKE, types.ServiceTypeSameDay,
}
statuses := []types.OrderStatus{
types.OrderStatusCreated, types.OrderStatusAwaitingPickup, types.OrderStatusPickedUp,
types.OrderStatusInTransit, types.OrderStatusDelivered, types.OrderStatusFailed,
types.OrderStatusCancelled,
}

rapid.Check(t, func(t *rapid.T) {
order := &types.Order{
OrderID:          rapid.StringMatching(`[a-z0-9\-]{8,16}`).Draw(t, "order_id"),
TrackingNumber:   rapid.StringMatching(`[A-Z0-9]{16}`).Draw(t, "tracking_number"),
SenderUserID:     rapid.StringMatching(`[a-z0-9]{4,12}`).Draw(t, "sender_user_id"),
SenderName:       rapid.StringMatching(`[A-Za-z]{2,20}`).Draw(t, "sender_name"),
SenderAddress:    rapid.StringMatching(`[A-Za-z0-9]{5,30}`).Draw(t, "sender_address"),
SenderPhone:      rapid.StringMatching(`08[0-9]{8,10}`).Draw(t, "sender_phone"),
SenderCityCode:   rapid.StringMatching(`[A-Z]{3}`).Draw(t, "sender_city_code"),
ReceiverName:     rapid.StringMatching(`[A-Za-z]{2,20}`).Draw(t, "receiver_name"),
ReceiverAddress:  rapid.StringMatching(`[A-Za-z0-9]{5,30}`).Draw(t, "receiver_address"),
ReceiverPhone:    rapid.StringMatching(`08[0-9]{8,10}`).Draw(t, "receiver_phone"),
ReceiverCityCode: rapid.StringMatching(`[A-Z]{3}`).Draw(t, "receiver_city_code"),
Weight:           rapid.Float64Range(0.1, 100.0).Draw(t, "weight"),
Length:           rapid.Float64Range(1.0, 200.0).Draw(t, "length"),
Width:            rapid.Float64Range(1.0, 200.0).Draw(t, "width"),
Height:           rapid.Float64Range(1.0, 200.0).Draw(t, "height"),
ServiceType:      rapid.SampledFrom(serviceTypes).Draw(t, "service_type"),
Price:            rapid.Float64Range(1000.0, 1000000.0).Draw(t, "price"),
IsCOD:            rapid.Bool().Draw(t, "is_cod"),
CODAmount:        rapid.Float64Range(0.0, 500000.0).Draw(t, "cod_amount"),
Insurance:        rapid.Bool().Draw(t, "insurance"),
ItemDescription:  rapid.StringMatching(`[A-Za-z0-9]{3,50}`).Draw(t, "item_description"),
Status:           rapid.SampledFrom(statuses).Draw(t, "status"),
CreatedAt:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
UpdatedAt:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
}

db, mock, err := sqlmock.New()
require.NoError(t, err)
defer db.Close()

// Expect SaveOrder INSERT
mock.ExpectExec("INSERT INTO orders").
WithArgs(
order.OrderID, order.TrackingNumber, order.SenderUserID, order.SenderName,
order.SenderAddress, order.SenderPhone, order.SenderCityCode,
order.ReceiverName, order.ReceiverAddress, order.ReceiverPhone, order.ReceiverCityCode,
order.Weight, order.Length, order.Width, order.Height,
string(order.ServiceType), order.Price, order.IsCOD, order.CODAmount, order.Insurance,
order.ItemDescription, string(order.Status), order.CreatedAt, order.UpdatedAt,
).
WillReturnResult(sqlmock.NewResult(1, 1))

// Expect FindOrderByID SELECT returning the same data
rows := sqlmock.NewRows(columns)
addOrderRow(rows, order)
mock.ExpectQuery("SELECT").
WithArgs(order.OrderID).
WillReturnRows(rows)

repo := repository.NewOrderRepository(db)

// Save
err = repo.SaveOrder(context.Background(), order)
require.NoError(t, err)

// Retrieve
got, err := repo.FindOrderByID(context.Background(), order.OrderID)
require.NoError(t, err)
require.NotNil(t, got)

// All fields must match
assert.Equal(t, order.OrderID, got.OrderID)
assert.Equal(t, order.TrackingNumber, got.TrackingNumber)
assert.Equal(t, order.SenderUserID, got.SenderUserID)
assert.Equal(t, order.SenderName, got.SenderName)
assert.Equal(t, order.SenderAddress, got.SenderAddress)
assert.Equal(t, order.SenderPhone, got.SenderPhone)
assert.Equal(t, order.SenderCityCode, got.SenderCityCode)
assert.Equal(t, order.ReceiverName, got.ReceiverName)
assert.Equal(t, order.ReceiverAddress, got.ReceiverAddress)
assert.Equal(t, order.ReceiverPhone, got.ReceiverPhone)
assert.Equal(t, order.ReceiverCityCode, got.ReceiverCityCode)
assert.Equal(t, order.Weight, got.Weight)
assert.Equal(t, order.Length, got.Length)
assert.Equal(t, order.Width, got.Width)
assert.Equal(t, order.Height, got.Height)
assert.Equal(t, order.ServiceType, got.ServiceType)
assert.Equal(t, order.Price, got.Price)
assert.Equal(t, order.IsCOD, got.IsCOD)
assert.Equal(t, order.CODAmount, got.CODAmount)
assert.Equal(t, order.Insurance, got.Insurance)
assert.Equal(t, order.ItemDescription, got.ItemDescription)
assert.Equal(t, order.Status, got.Status)
assert.Equal(t, order.CreatedAt, got.CreatedAt)
assert.Equal(t, order.UpdatedAt, got.UpdatedAt)

require.NoError(t, mock.ExpectationsWereMet())
})
}

// ─── Task 16.6: Property 10 — Dynamic Filter Query Construction ───────────────
// Validates: Requirements 11.3

func TestProperty10_DynamicFilterQueryConstruction(t *testing.T) {
statuses := []types.OrderStatus{
types.OrderStatusCreated, types.OrderStatusAwaitingPickup, types.OrderStatusPickedUp,
types.OrderStatusInTransit, types.OrderStatusDelivered, types.OrderStatusFailed,
types.OrderStatusCancelled,
}

rapid.Check(t, func(t *rapid.T) {
// Generate random non-empty filter combinations
hasUserID := rapid.Bool().Draw(t, "has_user_id")
hasStatus := rapid.Bool().Draw(t, "has_status")

// Ensure at least one filter is set
if !hasUserID && !hasStatus {
hasUserID = true
}

var userID string
var status types.OrderStatus

if hasUserID {
userID = rapid.StringMatching(`[a-z0-9]{4,12}`).Draw(t, "user_id")
}
if hasStatus {
status = rapid.SampledFrom(statuses).Draw(t, "status")
}

page := rapid.IntRange(1, 10).Draw(t, "page")
limit := rapid.IntRange(1, 50).Draw(t, "limit")

db, mock, err := sqlmock.New()
require.NoError(t, err)
defer db.Close()

// Build expected args as driver.Value slice (required by go-sqlmock v1)
var expectedArgs []driver.Value
if hasUserID {
expectedArgs = append(expectedArgs, userID)
}
if hasStatus {
expectedArgs = append(expectedArgs, string(status))
}
expectedArgs = append(expectedArgs, int64(limit), int64((page-1)*limit))

// Choose query pattern based on which filters are set
var queryPattern string
switch {
case hasUserID:
queryPattern = "WHERE sender_user_id"
case hasStatus:
queryPattern = "WHERE status"
}

rows := sqlmock.NewRows(columns)
mock.ExpectQuery(queryPattern).
WithArgs(expectedArgs...).
WillReturnRows(rows)

repo := repository.NewOrderRepository(db)
params := types.ListOrdersParams{
UserID: userID,
Status: status,
Page:   page,
Limit:  limit,
}

got, err := repo.FindOrders(context.Background(), params)
require.NoError(t, err)
assert.Empty(t, got)
require.NoError(t, mock.ExpectationsWereMet())
})
}
