package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"e4-api/internal/config"
	"e4-api/internal/db"
	"e4-api/internal/models"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

var jsonStoreKeyPattern = regexp.MustCompile(`^[A-Za-z0-9]+$`)

type JSONStoreHandler struct{}

type JSONStoreRateBucket struct {
	Count       int
	WindowStart time.Time
}

type JSONStoreListResponse struct {
	Items          []JSONStoreAdminItem `json:"items"`
	Total          int64                `json:"total"`
	TotalSizeBytes int64                `json:"total_size_bytes"`
	NewestKey      string               `json:"newest_key"`
	SortOrder      string               `json:"sort_order"`
}

type JSONStoreAdminItem struct {
	ID        uint      `json:"id"`
	Key       string    `json:"key"`
	SizeBytes int64     `json:"size_bytes"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	IsExpired bool      `json:"is_expired"`
}

type JSONStoreAdminContentResponse struct {
	Key       string    `json:"key"`
	Content   string    `json:"content"`
	SizeBytes int64     `json:"size_bytes"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	IsExpired bool      `json:"is_expired"`
}

var (
	jsonStoreRateMu       sync.Mutex
	jsonStoreReadBuckets  = make(map[string]*JSONStoreRateBucket)
	jsonStoreWriteBuckets = make(map[string]*JSONStoreRateBucket)
)

func NewJSONStoreHandler() *JSONStoreHandler {
	return &JSONStoreHandler{}
}

func (h *JSONStoreHandler) Create(c echo.Context) error {
	return h.upsert(c, false)
}

func (h *JSONStoreHandler) Upsert(c echo.Context) error {
	return h.upsert(c, true)
}

func (h *JSONStoreHandler) Get(c echo.Context) error {
	if !allowJSONStoreRequest(c, false) {
		return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "请求过于频繁，请稍后重试"})
	}

	key, ok := validateJSONStoreKey(c.Param("key"))
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "key 只能包含字母和数字，长度必须符合限制"})
	}

	item, found, err := loadJSONStoreItem(key)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "读取 JSON 失败"})
	}
	if !found {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "JSON 不存在或已过期"})
	}

	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
	c.Response().Header().Set(echo.HeaderCacheControl, "no-store")
	return c.Blob(http.StatusOK, echo.MIMEApplicationJSONCharsetUTF8, []byte(item.Content))
}

func (h *JSONStoreHandler) Delete(c echo.Context) error {
	if !allowJSONStoreRequest(c, true) {
		return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "请求过于频繁，请稍后重试"})
	}

	key, ok := validateJSONStoreKey(c.Param("key"))
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "key 只能包含字母和数字，长度必须符合限制"})
	}

	if err := db.DB.Where("key = ?", key).Delete(&models.JSONStoreItem{}).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "删除 JSON 失败"})
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *JSONStoreHandler) AdminList(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	search := strings.TrimSpace(c.QueryParam("search"))
	sortOrder := normalizeJSONStoreSortOrder(c.QueryParam("sort"))

	buildQuery := func() *gorm.DB {
		query := db.DB.Model(&models.JSONStoreItem{})
		if search != "" {
			searchPattern := "%" + search + "%"
			query = query.Where("CAST(id AS TEXT) LIKE ? OR key LIKE ? OR content LIKE ?", searchPattern, searchPattern, searchPattern)
		}
		return query
	}

	var total int64
	if err := buildQuery().Count(&total).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "获取 JSON 列表失败"})
	}

	var totalSizeBytes int64
	if err := buildQuery().Select("COALESCE(SUM(size_bytes), 0)").Scan(&totalSizeBytes).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "获取 JSON 汇总失败"})
	}

	var newestItem models.JSONStoreItem
	newestKey := ""
	if err := buildQuery().Order("created_at DESC, id DESC").Limit(1).First(&newestItem).Error; err == nil {
		newestKey = newestItem.Key
	} else if err != nil && err != gorm.ErrRecordNotFound {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "获取 JSON 汇总失败"})
	}

	var items []models.JSONStoreItem
	offset := (page - 1) * perPage
	if err := buildQuery().Order(buildJSONStoreSortClause(sortOrder)).Offset(offset).Limit(perPage).Find(&items).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "获取 JSON 列表失败"})
	}

	now := time.Now()
	response := make([]JSONStoreAdminItem, 0, len(items))
	for _, item := range items {
		response = append(response, JSONStoreAdminItem{
			ID:        item.ID,
			Key:       item.Key,
			SizeBytes: item.SizeBytes,
			ExpiresAt: item.ExpiresAt,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
			IsExpired: item.ExpiresAt.Before(now),
		})
	}

	return c.JSON(http.StatusOK, JSONStoreListResponse{
		Items:          response,
		Total:          total,
		TotalSizeBytes: totalSizeBytes,
		NewestKey:      newestKey,
		SortOrder:      sortOrder,
	})
}

func (h *JSONStoreHandler) AdminGetContent(c echo.Context) error {
	key, ok := validateJSONStoreKey(c.Param("key"))
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "key 只能包含字母和数字，长度必须符合限制"})
	}

	var item models.JSONStoreItem
	if err := db.DB.First(&item, "key = ?", key).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "JSON 不存在"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "读取 JSON 内容失败"})
	}

	return c.JSON(http.StatusOK, JSONStoreAdminContentResponse{
		Key:       item.Key,
		Content:   item.Content,
		SizeBytes: item.SizeBytes,
		ExpiresAt: item.ExpiresAt,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
		IsExpired: item.ExpiresAt.Before(time.Now()),
	})
}

func (h *JSONStoreHandler) AdminDelete(c echo.Context) error {
	return h.Delete(c)
}

func (h *JSONStoreHandler) upsert(c echo.Context, allowOverwrite bool) error {
	if !allowJSONStoreRequest(c, true) {
		return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "请求过于频繁，请稍后重试"})
	}

	key, ok := validateJSONStoreKey(c.Param("key"))
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "key 只能包含字母和数字，长度必须符合限制"})
	}

	body, err := readJSONStoreBody(c)
	if err != nil {
		status, message := jsonStoreHTTPError(err, http.StatusBadRequest, "读取 JSON 失败")
		return c.JSON(status, map[string]string{"error": message})
	}

	expiresAt, err := resolveJSONStoreExpiry(c.QueryParam("ttl_days"))
	if err != nil {
		status, message := jsonStoreHTTPError(err, http.StatusBadRequest, "ttl_days 无效")
		return c.JSON(status, map[string]string{"error": message})
	}

	var existing models.JSONStoreItem
	lookupErr := db.DB.First(&existing, "key = ?", key).Error
	if lookupErr != nil && lookupErr != gorm.ErrRecordNotFound {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "保存 JSON 失败"})
	}

	if lookupErr == nil && !allowOverwrite && existing.ExpiresAt.After(time.Now()) {
		return c.JSON(http.StatusConflict, map[string]string{"error": "key 已存在"})
	}

	if err := ensureJSONStoreCapacity(body, lookupErr == nil, existing.SizeBytes); err != nil {
		status, message := jsonStoreHTTPError(err, http.StatusInsufficientStorage, "JSON 存储空间不足")
		return c.JSON(status, map[string]string{"error": message})
	}

	now := time.Now()
	item := models.JSONStoreItem{
		Key:       key,
		Content:   string(body),
		SizeBytes: int64(len(body)),
		ExpiresAt: expiresAt,
	}
	status := http.StatusCreated

	if lookupErr == nil {
		item.ID = existing.ID
		item.CreatedAt = existing.CreatedAt
		item.UpdatedAt = now
		status = http.StatusOK
	} else {
		item.CreatedAt = now
		item.UpdatedAt = now
	}

	if err := db.DB.Save(&item).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "保存 JSON 失败"})
	}

	return c.JSON(status, map[string]interface{}{
		"key":        item.Key,
		"size_bytes": item.SizeBytes,
		"expires_at": item.ExpiresAt,
		"created_at": item.CreatedAt,
		"updated_at": item.UpdatedAt,
	})
}

var errJSONStoreTooLarge = echo.NewHTTPError(http.StatusRequestEntityTooLarge, "JSON 大小超过限制")

func readJSONStoreBody(c echo.Context) ([]byte, error) {
	maxSize := config.Cfg.JSONStore.MaxSizeBytes
	reader := http.MaxBytesReader(c.Response(), c.Request().Body, maxSize)
	defer reader.Close()

	body, err := io.ReadAll(reader)
	if err != nil {
		if strings.Contains(err.Error(), "http: request body too large") {
			return nil, errJSONStoreTooLarge
		}
		return nil, echo.NewHTTPError(http.StatusBadRequest, "读取 JSON 失败")
	}
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "JSON 内容不能为空")
	}
	if !json.Valid(body) {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "提交内容不是合法 JSON")
	}
	compactBody := new(bytes.Buffer)
	if err := json.Compact(compactBody, body); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "提交内容不是合法 JSON")
	}
	if int64(compactBody.Len()) > maxSize {
		return nil, errJSONStoreTooLarge
	}
	return compactBody.Bytes(), nil
}

func resolveJSONStoreExpiry(ttlParam string) (time.Time, error) {
	ttlDays := config.Cfg.JSONStore.DefaultTTLDays
	if strings.TrimSpace(ttlParam) != "" {
		value, err := strconv.Atoi(ttlParam)
		if err != nil || value <= 0 {
			return time.Time{}, echo.NewHTTPError(http.StatusBadRequest, "ttl_days 必须是正整数")
		}
		ttlDays = value
	}
	if ttlDays > config.Cfg.JSONStore.MaxTTLDays {
		return time.Time{}, echo.NewHTTPError(http.StatusBadRequest, "ttl_days 超过允许的最大天数")
	}
	return time.Now().AddDate(0, 0, ttlDays), nil
}

func validateJSONStoreKey(key string) (string, bool) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", false
	}
	if len(key) < config.Cfg.JSONStore.MinKeyLength || len(key) > config.Cfg.JSONStore.MaxKeyLength {
		return "", false
	}
	if !jsonStoreKeyPattern.MatchString(key) {
		return "", false
	}
	return key, true
}

func loadJSONStoreItem(key string) (*models.JSONStoreItem, bool, error) {
	var item models.JSONStoreItem
	if err := db.DB.First(&item, "key = ?", key).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}
	if item.ExpiresAt.Before(time.Now()) {
		return nil, false, nil
	}
	return &item, true, nil
}

func ensureJSONStoreCapacity(content []byte, exists bool, existingSize int64) error {
	var totalItems int64
	if err := db.DB.Model(&models.JSONStoreItem{}).Count(&totalItems).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "检查存储容量失败")
	}
	if !exists && totalItems >= config.Cfg.JSONStore.MaxItems {
		return echo.NewHTTPError(http.StatusInsufficientStorage, "JSON 存储条目已满")
	}

	var totalBytes int64
	if err := db.DB.Model(&models.JSONStoreItem{}).Select("COALESCE(SUM(size_bytes), 0)").Scan(&totalBytes).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "检查存储容量失败")
	}
	if totalBytes-existingSize+int64(len(content)) > config.Cfg.JSONStore.MaxTotalBytes {
		return echo.NewHTTPError(http.StatusInsufficientStorage, "JSON 存储空间不足")
	}
	return nil
}

func allowJSONStoreRequest(c echo.Context, write bool) bool {
	clientIP := getClientIP(c)
	window := time.Duration(config.Cfg.JSONStore.RateLimitWindowSeconds) * time.Second
	if window <= 0 {
		window = 60 * time.Second
	}
	limit := config.Cfg.JSONStore.ReadRateLimit
	buckets := jsonStoreReadBuckets
	if write {
		limit = config.Cfg.JSONStore.WriteRateLimit
		buckets = jsonStoreWriteBuckets
	}
	if limit <= 0 {
		return true
	}

	jsonStoreRateMu.Lock()
	defer jsonStoreRateMu.Unlock()

	now := time.Now()
	bucket, exists := buckets[clientIP]
	if !exists || now.Sub(bucket.WindowStart) >= window {
		buckets[clientIP] = &JSONStoreRateBucket{Count: 1, WindowStart: now}
		return true
	}
	if bucket.Count >= limit {
		return false
	}
	bucket.Count++
	return true
}

func jsonStoreHTTPError(err error, fallbackStatus int, fallbackMessage string) (int, string) {
	if httpErr, ok := err.(*echo.HTTPError); ok {
		message, ok := httpErr.Message.(string)
		if !ok || strings.TrimSpace(message) == "" {
			message = fallbackMessage
		}
		if httpErr.Code == 0 {
			return fallbackStatus, message
		}
		return httpErr.Code, message
	}
	return fallbackStatus, fallbackMessage
}

func normalizeJSONStoreSortOrder(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), "asc") {
		return "asc"
	}
	return "desc"
}

func buildJSONStoreSortClause(sortOrder string) string {
	if sortOrder == "asc" {
		return "updated_at ASC, id ASC"
	}
	return "updated_at DESC, id DESC"
}
