package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withParam(req *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// ---------------- Subscribe ----------------

func TestSubscribe_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT EXISTS(SELECT 1 FROM subscriptions WHERE email=$1 AND city=$2)")).
		WithArgs("u@x.com", "Lviv").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	mock.ExpectExec(regexp.QuoteMeta(
		"INSERT INTO subscriptions (id, email, city, frequency, token, confirmed, created_at)")).
		WithArgs(
			sqlmock.AnyArg(),    
			"u@x.com",           
			"Lviv",              
			"hourly",            
			sqlmock.AnyArg(),    
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	h := &handler{db: db}
	req := httptest.NewRequest(http.MethodPost, "/api/subscribe",
		strings.NewReader(`{"email":"u@x.com","city":"Lviv","frequency":"hourly"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Subscribe(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var res subscribeRes
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &res))
	assert.Contains(t, res.Message, "Subscription successful")
	assert.Regexp(t, `^[0-9a-fA-F-]{36}$`, res.Token)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSubscribe_Conflict(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT EXISTS(SELECT 1 FROM subscriptions WHERE email=$1 AND city=$2)")).
		WithArgs("dup@x.com", "Kyiv").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	h := &handler{db: db}
	req := httptest.NewRequest(http.MethodPost, "/api/subscribe",
		strings.NewReader(`{"email":"dup@x.com","city":"Kyiv","frequency":"daily"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Subscribe(rr, req)

	assert.Equal(t, http.StatusConflict, rr.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

// ---------------- ConfirmSubscription ----------------

func TestConfirmSubscription_InvalidToken(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	h := &handler{db: db}
	req := httptest.NewRequest(http.MethodGet, "/api/confirm/bad", nil)
	req = withParam(req, "token", "bad")
	rr := httptest.NewRecorder()

	h.ConfirmSubscription(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestConfirmSubscription_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	token := "00000000-0000-0000-0000-000000000000"
	mock.ExpectExec(regexp.QuoteMeta(
		"UPDATE subscriptions SET confirmed=true WHERE token=$1")).
		WithArgs(token).
		WillReturnResult(sqlmock.NewResult(0, 0))

	h := &handler{db: db}
	req := httptest.NewRequest(http.MethodGet, "/api/confirm/"+token, nil)
	req = withParam(req, "token", token)
	rr := httptest.NewRecorder()

	h.ConfirmSubscription(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestConfirmSubscription_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	token := "11111111-1111-1111-1111-111111111111"
	mock.ExpectExec(regexp.QuoteMeta(
		"UPDATE subscriptions SET confirmed=true WHERE token=$1")).
		WithArgs(token).
		WillReturnResult(sqlmock.NewResult(1, 1))

	h := &handler{db: db}
	req := httptest.NewRequest(http.MethodGet, "/api/confirm/"+token, nil)
	req = withParam(req, "token", token)
	rr := httptest.NewRecorder()

	h.ConfirmSubscription(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

// ---------------- Unsubscribe ----------------

func TestUnsubscribe_InvalidToken(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	h := &handler{db: db}
	req := httptest.NewRequest(http.MethodGet, "/api/unsubscribe/bad", nil)
	req = withParam(req, "token", "bad")
	rr := httptest.NewRecorder()

	h.Unsubscribe(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUnsubscribe_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	token := "22222222-2222-2222-2222-222222222222"
	mock.ExpectExec(regexp.QuoteMeta(
		"DELETE FROM subscriptions WHERE token=$1")).
		WithArgs(token).
		WillReturnResult(sqlmock.NewResult(0, 0))

	h := &handler{db: db}
	req := httptest.NewRequest(http.MethodGet, "/api/unsubscribe/"+token, nil)
	req = withParam(req, "token", token)
	rr := httptest.NewRecorder()

	h.Unsubscribe(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUnsubscribe_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	token := "33333333-3333-3333-3333-333333333333"
	mock.ExpectExec(regexp.QuoteMeta(
		"DELETE FROM subscriptions WHERE token=$1")).
		WithArgs(token).
		WillReturnResult(sqlmock.NewResult(1, 1))

	h := &handler{db: db}
	req := httptest.NewRequest(http.MethodGet, "/api/unsubscribe/"+token, nil)
	req = withParam(req, "token", token)
	rr := httptest.NewRecorder()

	h.Unsubscribe(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}
