package handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"e4-api/internal/config"
	"e4-api/internal/db"
	"e4-api/internal/models"

	"github.com/glebarez/sqlite"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestJSONStoreHandler(t *testing.T) (*JSONStoreHandler, *echo.Echo) {
	t.Helper()

	dsn := fmt.Sprintf("file:json-store-test-%d?mode=memory&cache=shared", time.Now().UnixNano())
	database, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&models.JSONStoreItem{}))

	db.DB = database
	config.Cfg = &config.Config{
		JSONStore: config.JSONStoreConfig{
			MaxSizeBytes:           512 * 1024,
			DefaultTTLDays:         30,
			MaxTTLDays:             90,
			MinKeyLength:           6,
			MaxKeyLength:           64,
			MaxItems:               1000,
			MaxTotalBytes:          128 * 1024 * 1024,
			ReadRateLimit:          1000,
			WriteRateLimit:         1000,
			RateLimitWindowSeconds: 60,
		},
	}
	jsonStoreReadBuckets = map[string]*JSONStoreRateBucket{}
	jsonStoreWriteBuckets = map[string]*JSONStoreRateBucket{}

	return NewJSONStoreHandler(), echo.New()
}

func TestJSONStoreCreateAndGet(t *testing.T) {
	handler, e := setupTestJSONStoreHandler(t)
	body := bytes.NewBufferString(`{"message":"hello"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/json/Abc123", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/json/:key")
	c.SetParamNames("key")
	c.SetParamValues("Abc123")

	err := handler.Create(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), `"key":"Abc123"`)
	assert.Contains(t, rec.Body.String(), `"size_bytes":19`)

	getReq := httptest.NewRequest(http.MethodGet, "/api/json/Abc123", nil)
	getRec := httptest.NewRecorder()
	getCtx := e.NewContext(getReq, getRec)
	getCtx.SetPath("/api/json/:key")
	getCtx.SetParamNames("key")
	getCtx.SetParamValues("Abc123")

	err = handler.Get(getCtx)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, getRec.Code)
	assert.JSONEq(t, `{"message":"hello"}`, getRec.Body.String())
	assert.Equal(t, "no-store", getRec.Header().Get(echo.HeaderCacheControl))
}

func TestJSONStoreCreateRejectsConflict(t *testing.T) {
	handler, e := setupTestJSONStoreHandler(t)
	seedJSONStoreItems(t, []models.JSONStoreItem{{Key: "Abc123", Content: `{"ok":true}`, SizeBytes: 11, ExpiresAt: time.Now().Add(24 * time.Hour)}})

	req := httptest.NewRequest(http.MethodPost, "/api/json/Abc123", bytes.NewBufferString(`{"ok":false}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/json/:key")
	c.SetParamNames("key")
	c.SetParamValues("Abc123")

	err := handler.Create(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusConflict, rec.Code)
	assert.JSONEq(t, `{"error":"key 已存在"}`, rec.Body.String())
}

func TestJSONStoreUpsertRefreshesExpiry(t *testing.T) {
	handler, e := setupTestJSONStoreHandler(t)
	oldExpiry := time.Now().Add(24 * time.Hour)
	seedJSONStoreItems(t, []models.JSONStoreItem{{Key: "Abc123", Content: `{"ok":true}`, SizeBytes: 11, ExpiresAt: oldExpiry}})

	req := httptest.NewRequest(http.MethodPut, "/api/json/Abc123?ttl_days=45", bytes.NewBufferString(`{"ok":false}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/json/:key")
	c.SetParamNames("key")
	c.SetParamValues("Abc123")

	err := handler.Upsert(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var item models.JSONStoreItem
	require.NoError(t, db.DB.First(&item, "key = ?", "Abc123").Error)
	assert.JSONEq(t, `{"ok":false}`, item.Content)
	assert.True(t, item.ExpiresAt.After(oldExpiry))
	assert.WithinDuration(t, time.Now().AddDate(0, 0, 45), item.ExpiresAt, 2*time.Minute)
}

func TestJSONStoreGetTreatsExpiredAsNotFound(t *testing.T) {
	handler, e := setupTestJSONStoreHandler(t)
	seedJSONStoreItems(t, []models.JSONStoreItem{{Key: "Abc123", Content: `{"old":true}`, SizeBytes: 12, ExpiresAt: time.Now().Add(-time.Hour)}})

	req := httptest.NewRequest(http.MethodGet, "/api/json/Abc123", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/json/:key")
	c.SetParamNames("key")
	c.SetParamValues("Abc123")

	err := handler.Get(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, rec.Code)
	assert.JSONEq(t, `{"error":"JSON 不存在或已过期"}`, rec.Body.String())
}

func TestJSONStoreRejectsInvalidJSON(t *testing.T) {
	handler, e := setupTestJSONStoreHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/json/Abc123", bytes.NewBufferString(`{"a":`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/json/:key")
	c.SetParamNames("key")
	c.SetParamValues("Abc123")

	err := handler.Create(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.JSONEq(t, `{"error":"提交内容不是合法 JSON"}`, rec.Body.String())
}

func TestJSONStoreRejectsOversizedBody(t *testing.T) {
	handler, e := setupTestJSONStoreHandler(t)
	config.Cfg.JSONStore.MaxSizeBytes = 16
	req := httptest.NewRequest(http.MethodPost, "/api/json/Abc123", bytes.NewBufferString(`{"value":"1234567890abcdef"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/json/:key")
	c.SetParamNames("key")
	c.SetParamValues("Abc123")

	err := handler.Create(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	assert.JSONEq(t, `{"error":"JSON 大小超过限制"}`, rec.Body.String())
}

func TestJSONStoreRejectsTTLAboveLimit(t *testing.T) {
	handler, e := setupTestJSONStoreHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/json/Abc123?ttl_days=120", bytes.NewBufferString(`{"ok":true}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/json/:key")
	c.SetParamNames("key")
	c.SetParamValues("Abc123")

	err := handler.Create(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.JSONEq(t, `{"error":"ttl_days 超过允许的最大天数"}`, rec.Body.String())
}

func TestJSONStoreRejectsInvalidKey(t *testing.T) {
	handler, e := setupTestJSONStoreHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/json/abc-123", bytes.NewBufferString(`{"ok":true}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/json/:key")
	c.SetParamNames("key")
	c.SetParamValues("abc-123")

	err := handler.Create(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.JSONEq(t, `{"error":"key 只能包含字母和数字，长度必须符合限制"}`, rec.Body.String())
}

func TestJSONStoreAdminListAndContent(t *testing.T) {
	handler, e := setupTestJSONStoreHandler(t)
	seedJSONStoreItems(t, []models.JSONStoreItem{
		{Key: "Abc123", Content: `{"a":1}`, SizeBytes: 7, ExpiresAt: time.Now().Add(24 * time.Hour), UpdatedAt: time.Now().Add(-time.Hour)},
		{Key: "Xyz789", Content: `{"b":2}`, SizeBytes: 7, ExpiresAt: time.Now().Add(-time.Hour), UpdatedAt: time.Now()},
	})

	listReq := httptest.NewRequest(http.MethodGet, "/api/admin/json?page=1&per_page=20", nil)
	listRec := httptest.NewRecorder()
	listCtx := e.NewContext(listReq, listRec)

	err := handler.AdminList(listCtx)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, listRec.Code)
	assert.Contains(t, listRec.Body.String(), `"total":2`)
	assert.Contains(t, listRec.Body.String(), `"total_size_bytes":14`)
	assert.Contains(t, listRec.Body.String(), `"sort_order":"desc"`)
	assert.Contains(t, listRec.Body.String(), `"key":"Xyz789"`)
	assert.Contains(t, listRec.Body.String(), `"is_expired":true`)

	contentReq := httptest.NewRequest(http.MethodGet, "/api/admin/json/Abc123/content", nil)
	contentRec := httptest.NewRecorder()
	contentCtx := e.NewContext(contentReq, contentRec)
	contentCtx.SetPath("/api/admin/json/:key/content")
	contentCtx.SetParamNames("key")
	contentCtx.SetParamValues("Abc123")

	err = handler.AdminGetContent(contentCtx)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, contentRec.Code)
	assert.Contains(t, contentRec.Body.String(), `"content":"{\"a\":1}"`)
}

func TestJSONStoreAdminListSupportsSearchAndAscendingSort(t *testing.T) {
	handler, e := setupTestJSONStoreHandler(t)
	older := time.Now().Add(-2 * time.Hour)
	newer := time.Now().Add(-time.Hour)
	seedJSONStoreItems(t, []models.JSONStoreItem{
		{Key: "Abc123", Content: `{"profile":"alpha"}`, SizeBytes: 19, ExpiresAt: time.Now().Add(24 * time.Hour), UpdatedAt: older},
		{Key: "Xyz789", Content: `{"profile":"beta-search"}`, SizeBytes: 25, ExpiresAt: time.Now().Add(24 * time.Hour), UpdatedAt: newer},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/admin/json?page=1&per_page=20&search=beta&sort=asc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.AdminList(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"total":1`)
	assert.Contains(t, rec.Body.String(), `"total_size_bytes":25`)
	assert.Contains(t, rec.Body.String(), `"sort_order":"asc"`)
	assert.Contains(t, rec.Body.String(), `"newest_key":"Xyz789"`)
	assert.Contains(t, rec.Body.String(), `"key":"Xyz789"`)
	assert.NotContains(t, rec.Body.String(), `"key":"Abc123"`)
}

func TestJSONStoreAdminDelete(t *testing.T) {
	handler, e := setupTestJSONStoreHandler(t)
	seedJSONStoreItems(t, []models.JSONStoreItem{{Key: "Abc123", Content: `{"a":1}`, SizeBytes: 7, ExpiresAt: time.Now().Add(24 * time.Hour)}})

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/json/Abc123", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/admin/json/:key")
	c.SetParamNames("key")
	c.SetParamValues("Abc123")

	err := handler.AdminDelete(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, rec.Code)

	var count int64
	require.NoError(t, db.DB.Model(&models.JSONStoreItem{}).Where("key = ?", "Abc123").Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func seedJSONStoreItems(t *testing.T, items []models.JSONStoreItem) {
	t.Helper()
	for _, item := range items {
		require.NoError(t, db.DB.Create(&item).Error)
	}
}
