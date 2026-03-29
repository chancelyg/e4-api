package handlers

import (
	"bytes"
	"e4-api/internal/models"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"e4-api/internal/db"
)

func setupTestDiaryHandler(t *testing.T) (*DiaryHandler, *echo.Echo) {
	t.Helper()

	dsn := fmt.Sprintf("file:diary-test-%d?mode=memory&cache=shared", time.Now().UnixNano())
	database, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&models.Diary{}))

	db.DB = database

	return NewDiaryHandler(), echo.New()
}

func seedDiaries(t *testing.T, diaries []models.Diary) {
	t.Helper()
	for _, diary := range diaries {
		require.NoError(t, db.DB.Create(&diary).Error)
	}
}

func TestCalculateStats(t *testing.T) {
	tests := []struct {
		name     string
		diaries  []models.Diary
		expected DiaryStats
	}{
		{
			name:     "empty diaries",
			diaries:  []models.Diary{},
			expected: DiaryStats{},
		},
		{
			name: "single diary",
			diaries: []models.Diary{
				{CreateDate: "2024-01-01"},
			},
			expected: DiaryStats{
				TotalCount:         1,
				MaxConsecutiveDays: 1,
				StartDate:          "2024-01-01",
				EndDate:            "2024-01-01",
				TimeSpan:           0,
			},
		},
		{
			name: "consecutive diaries",
			diaries: []models.Diary{
				{CreateDate: "2024-01-01"},
				{CreateDate: "2024-01-02"},
				{CreateDate: "2024-01-03"},
			},
			expected: DiaryStats{
				TotalCount:         3,
				MaxConsecutiveDays: 3,
				StartDate:          "2024-01-01",
				EndDate:            "2024-01-03",
				TimeSpan:           2,
			},
		},
		{
			name: "non-consecutive diaries",
			diaries: []models.Diary{
				{CreateDate: "2024-01-01"},
				{CreateDate: "2024-01-03"},
				{CreateDate: "2024-01-05"},
			},
			expected: DiaryStats{
				TotalCount:         3,
				MaxConsecutiveDays: 1,
				StartDate:          "2024-01-01",
				EndDate:            "2024-01-05",
				TimeSpan:           4,
			},
		},
		{
			name: "mixed consecutive and non-consecutive",
			diaries: []models.Diary{
				{CreateDate: "2024-01-01"},
				{CreateDate: "2024-01-02"},
				{CreateDate: "2024-01-04"},
				{CreateDate: "2024-01-05"},
				{CreateDate: "2024-01-06"},
			},
			expected: DiaryStats{
				TotalCount:         5,
				MaxConsecutiveDays: 3,
				StartDate:          "2024-01-01",
				EndDate:            "2024-01-06",
				TimeSpan:           5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateStats(tt.diaries)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateStatsConsecutiveEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		diaries  []models.Diary
		expected int
	}{
		{
			name: "gap in middle, consecutive at end",
			diaries: []models.Diary{
				{CreateDate: "2024-01-01"},
				{CreateDate: "2024-01-05"},
				{CreateDate: "2024-01-06"},
				{CreateDate: "2024-01-07"},
			},
			expected: 3,
		},
		{
			name: "gap at end, consecutive at start",
			diaries: []models.Diary{
				{CreateDate: "2024-01-01"},
				{CreateDate: "2024-01-02"},
				{CreateDate: "2024-01-03"},
				{CreateDate: "2024-01-10"},
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateStats(tt.diaries)
			assert.Equal(t, tt.expected, result.MaxConsecutiveDays)
		})
	}
}

func TestDiaryListSupportsKeywordAndDateFilters(t *testing.T) {
	handler, e := setupTestDiaryHandler(t)
	seedDiaries(t, []models.Diary{
		{Content: "今天 学习 Go 并写测试", CreateDate: "2024-01-05"},
		{Content: "学习 Svelte 页面布局", CreateDate: "2024-01-09"},
		{Content: "Go 搜索逻辑修复", CreateDate: "2024-02-01"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/diary?page=1&per_page=20&search=Go%20%E5%AD%A6%E4%B9%A0&start_date=2024-01-01&end_date=2024-01-31", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.List(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var response DiaryListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response.Diaries, 1)
	assert.Equal(t, int64(1), response.Total)
	assert.Equal(t, "今天 学习 Go 并写测试", response.Diaries[0].Content)
	assert.Equal(t, "2024-01-05", response.Diaries[0].CreateDate)
}

func TestDiaryListSupportsFuzzySearch(t *testing.T) {
	handler, e := setupTestDiaryHandler(t)
	seedDiaries(t, []models.Diary{
		{Content: "今天整理登录刷新问题并补充会话签名", CreateDate: "2024-03-02"},
		{Content: "重新设计日记页面布局", CreateDate: "2024-03-03"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/diary?search=%E7%99%BB%E5%88%B7", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.List(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var response DiaryListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response.Diaries, 1)
	assert.Equal(t, int64(1), response.Total)
	assert.Contains(t, response.Diaries[0].Content, "登录刷新")
}

func TestDiaryCreateRejectsBlankContent(t *testing.T) {
	handler, e := setupTestDiaryHandler(t)
	body := bytes.NewBufferString(`{"content":"   ","create_date":"2024-01-10"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/diary", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Create(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.JSONEq(t, `{"error":"日记内容不能为空"}`, rec.Body.String())
}

func TestDiaryUpdateRejectsBlankContent(t *testing.T) {
	handler, e := setupTestDiaryHandler(t)
	seedDiaries(t, []models.Diary{{Content: "原始内容", CreateDate: "2024-01-10"}})

	body := bytes.NewBufferString(`{"content":"   "}`)
	req := httptest.NewRequest(http.MethodPut, "/api/diary/1", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := handler.Update(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.JSONEq(t, `{"error":"日记内容不能为空"}`, rec.Body.String())
}

func TestDiaryDeleteReturnsNotFoundForMissingEntry(t *testing.T) {
	handler, e := setupTestDiaryHandler(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/diary/999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("999")

	err := handler.Delete(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, rec.Code)
	assert.JSONEq(t, `{"error":"日记不存在"}`, rec.Body.String())
}
