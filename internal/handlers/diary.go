package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"e4-api/internal/db"
	"e4-api/internal/models"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type DiaryHandler struct{}

type CreateDiaryRequest struct {
	Content string `json:"content" validate:"required"`
}

type DiaryListResponse struct {
	Diaries []models.Diary `json:"diaries"`
	Total   int64          `json:"total"`
}

type DiaryStats struct {
	TotalCount         int    `json:"total_count"`
	MaxConsecutiveDays int    `json:"max_consecutive_days"`
	StartDate          string `json:"start_date"`
	EndDate            string `json:"end_date"`
	TimeSpan           int    `json:"time_span_days"`
}

func NewDiaryHandler() *DiaryHandler {
	return &DiaryHandler{}
}

func (h *DiaryHandler) List(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	search := strings.TrimSpace(c.QueryParam("search"))
	startDate := c.QueryParam("start_date")
	endDate := c.QueryParam("end_date")
	sortOrder := normalizeDiarySortOrder(c.QueryParam("sort"), search, startDate, endDate)

	query := db.DB.Model(&models.Diary{})

	if search != "" {
		groups := buildSearchPatternGroups(search)
		for _, patterns := range groups {
			if len(patterns) == 0 {
				continue
			}
			searchQuery := db.DB.Where("content LIKE ?", patterns[0])
			for _, pattern := range patterns[1:] {
				searchQuery = searchQuery.Or("content LIKE ?", pattern)
			}
			query = query.Where(searchQuery)
		}
	}
	if startDate != "" {
		query = query.Where("create_date >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("create_date <= ?", endDate)
	}

	var total int64
	if result := query.Count(&total); result.Error != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "获取日记列表失败",
		})
	}

	var diaries []models.Diary
	offset := (page - 1) * perPage
	if result := query.Order(sortOrder).Offset(offset).Limit(perPage).Find(&diaries); result.Error != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "获取日记列表失败",
		})
	}

	return c.JSON(http.StatusOK, DiaryListResponse{
		Diaries: diaries,
		Total:   total,
	})
}

func normalizeDiarySortOrder(rawSort, search, startDate, endDate string) string {
	hasFilter := strings.TrimSpace(search) != "" || startDate != "" || endDate != ""
	if !hasFilter {
		return "create_date DESC, id DESC"
	}

	sort := strings.ToLower(strings.TrimSpace(rawSort))
	if sort == "desc" {
		return "create_date DESC, id DESC"
	}

	return "create_date ASC, id ASC"
}

func (h *DiaryHandler) Get(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "无效的日记 ID",
		})
	}

	var diary models.Diary
	if result := db.DB.First(&diary, id); result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "日记不存在",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "获取日记失败",
		})
	}

	return c.JSON(http.StatusOK, diary)
}

func (h *DiaryHandler) Create(c echo.Context) error {
	req := new(CreateDiaryRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "无效的请求数据",
		})
	}

	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "日记内容不能为空",
		})
	}

	createDate, err := nextDiaryCreateDate()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "创建日记失败",
		})
	}

	diary := models.Diary{
		Content:    req.Content,
		CreateDate: createDate,
	}

	if result := db.DB.Create(&diary); result.Error != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "创建日记失败",
		})
	}

	return c.JSON(http.StatusCreated, diary)
}

func (h *DiaryHandler) Delete(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "无效的日记 ID",
		})
	}

	result := db.DB.Delete(&models.Diary{}, id)
	if result.Error != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "删除日记失败",
		})
	}
	if result.RowsAffected == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "日记不存在",
		})
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *DiaryHandler) Stats(c echo.Context) error {
	var diaries []models.Diary
	if result := db.DB.Order("create_date ASC").Find(&diaries); result.Error != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "获取统计失败",
		})
	}

	if len(diaries) == 0 {
		return c.JSON(http.StatusOK, DiaryStats{
			TotalCount:         0,
			MaxConsecutiveDays: 0,
			StartDate:          "",
			EndDate:            "",
			TimeSpan:           0,
		})
	}

	stats := calculateStats(diaries)
	return c.JSON(http.StatusOK, stats)
}

func calculateStats(diaries []models.Diary) DiaryStats {
	if len(diaries) == 0 {
		return DiaryStats{}
	}

	stats := DiaryStats{
		TotalCount: len(diaries),
		StartDate:  diaries[0].CreateDate,
		EndDate:    diaries[len(diaries)-1].CreateDate,
	}

	// Calculate time span
	start, _ := time.Parse("2006-01-02", stats.StartDate)
	end, _ := time.Parse("2006-01-02", stats.EndDate)
	stats.TimeSpan = int(end.Sub(start).Hours() / 24)

	// Calculate max consecutive days
	dateMap := make(map[string]bool)
	for _, d := range diaries {
		dateMap[d.CreateDate] = true
	}

	maxConsecutive := 0
	currentConsecutive := 0

	current := start
	for !current.After(end) {
		dateStr := current.Format("2006-01-02")
		if dateMap[dateStr] {
			currentConsecutive++
			if currentConsecutive > maxConsecutive {
				maxConsecutive = currentConsecutive
			}
		} else {
			currentConsecutive = 0
		}
		current = current.AddDate(0, 0, 1)
	}

	stats.MaxConsecutiveDays = maxConsecutive
	return stats
}

func buildSearchPatternGroups(search string) [][]string {
	trimmed := strings.TrimSpace(search)
	if trimmed == "" {
		return nil
	}

	keywords := strings.Fields(trimmed)
	if len(keywords) == 0 {
		keywords = []string{trimmed}
	}

	groups := make([][]string, 0, len(keywords))
	for _, keyword := range keywords {
		seen := make(map[string]struct{})
		patterns := make([]string, 0, 2)
		addPattern := func(value string) {
			if value == "" {
				return
			}
			if _, exists := seen[value]; exists {
				return
			}
			seen[value] = struct{}{}
			patterns = append(patterns, value)
		}

		addPattern("%" + keyword + "%")
		addPattern(buildFuzzyLikePattern(keyword))
		groups = append(groups, patterns)
	}

	if len(keywords) == 1 {
		return groups
	}

	return groups
}

func buildFuzzyLikePattern(value string) string {
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) <= 1 {
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(value)*2 + 2)
	builder.WriteByte('%')
	for _, r := range value {
		builder.WriteRune(r)
		builder.WriteByte('%')
	}
	return builder.String()
}

func nextDiaryCreateDate() (string, error) {
	var latestDiary models.Diary
	result := db.DB.Order("create_date DESC, id DESC").Limit(1).First(&latestDiary)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return time.Now().Format("2006-01-02"), nil
		}
		return "", result.Error
	}

	latestDate, err := time.Parse("2006-01-02", latestDiary.CreateDate)
	if err != nil {
		return "", err
	}

	return latestDate.AddDate(0, 0, 1).Format("2006-01-02"), nil
}
