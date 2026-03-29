package handlers

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"e4-api/internal/db"
	"e4-api/internal/models"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

const goalDateLayout = "2006-01-02"

type GoalHandler struct{}

type CreateGoalRequest struct {
	ReactivateID uint     `json:"reactivate_id"`
	Name         string   `json:"name"`
	GoalType     string   `json:"goal_type"`
	Unit         string   `json:"unit"`
	AnnualTarget *float64 `json:"annual_target"`
	WeeklyTarget *int     `json:"weekly_target"`
}

type UpdateGoalRequest struct {
	Name         *string  `json:"name"`
	Unit         *string  `json:"unit"`
	AnnualTarget *float64 `json:"annual_target"`
	WeeklyTarget *int     `json:"weekly_target"`
}

type UpsertGoalRecordRequest struct {
	Quantity *float64 `json:"quantity"`
}

type GoalDashboardResponse struct {
	AnchorDate     string                  `json:"anchor_date"`
	Range          string                  `json:"range"`
	RangeStartDate string                  `json:"range_start_date"`
	RangeEndDate   string                  `json:"range_end_date"`
	CheckinDate    string                  `json:"checkin_date"`
	CalendarMonth  string                  `json:"calendar_month"`
	Goals          []GoalDashboardItem     `json:"goals"`
	InactiveGoals  []models.Goal           `json:"inactive_goals"`
	CalendarDays   []GoalCalendarDay       `json:"calendar_days"`
	DayDetails     []GoalCalendarDayDetail `json:"day_details"`
}

type GoalDashboardItem struct {
	ID                         uint               `json:"id"`
	Name                       string             `json:"name"`
	GoalType                   string             `json:"goal_type"`
	Unit                       string             `json:"unit"`
	AnnualTarget               *float64           `json:"annual_target"`
	WeeklyTarget               *int               `json:"weekly_target"`
	RangeCompletedCount        int                `json:"range_completed_count"`
	RangeQuantityTotal         float64            `json:"range_quantity_total"`
	AnnualCompletedCount       int                `json:"annual_completed_count"`
	AnnualQuantityTotal        float64            `json:"annual_quantity_total"`
	AnnualRemainingValue       *float64           `json:"annual_remaining_value"`
	AnnualProgressPercent      *float64           `json:"annual_progress_percent"`
	CurrentWeekCompletedCount  int                `json:"current_week_completed_count"`
	CurrentWeekProgressPercent *float64           `json:"current_week_progress_percent"`
	CheckinRecord              *GoalRecordPayload `json:"checkin_record"`
}

type GoalRecordPayload struct {
	RecordDate  string   `json:"record_date"`
	IsCompleted bool     `json:"is_completed"`
	Quantity    *float64 `json:"quantity"`
}

type GoalCalendarDay struct {
	Date           string `json:"date"`
	CompletedGoals int    `json:"completed_goals"`
	TotalGoals     int    `json:"total_goals"`
	Intensity      int    `json:"intensity"`
}

type GoalCalendarDayDetail struct {
	Date           string                  `json:"date"`
	CompletedGoals int                     `json:"completed_goals"`
	TotalGoals     int                     `json:"total_goals"`
	Items          []GoalCalendarDayRecord `json:"items"`
}

type GoalCalendarDayRecord struct {
	GoalID      uint     `json:"goal_id"`
	Name        string   `json:"name"`
	GoalType    string   `json:"goal_type"`
	Unit        string   `json:"unit"`
	IsCompleted bool     `json:"is_completed"`
	Quantity    *float64 `json:"quantity"`
}

func NewGoalHandler() *GoalHandler {
	return &GoalHandler{}
}

func (h *GoalHandler) List(c echo.Context) error {
	goals, err := listActiveGoals()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "获取目标列表失败"})
	}

	return c.JSON(http.StatusOK, map[string][]models.Goal{"goals": goals})
}

func (h *GoalHandler) Create(c echo.Context) error {
	var req CreateGoalRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "无效的请求数据"})
	}

	goal, err := buildGoalFromCreate(req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if req.ReactivateID > 0 {
		reactivated, err := reactivateGoal(req.ReactivateID, goal)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return c.JSON(http.StatusNotFound, map[string]string{"error": "停用目标不存在"})
			}
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusCreated, reactivated)
	}

	if err := db.DB.Create(&goal).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "创建目标失败"})
	}

	return c.JSON(http.StatusCreated, goal)
}

func (h *GoalHandler) Update(c echo.Context) error {
	goal, err := findGoalByParam(c.Param("id"))
	if err != nil {
		return goalErrorResponse(c, err, "目标不存在", "无效的目标 ID", "获取目标失败")
	}

	var req UpdateGoalRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "无效的请求数据"})
	}

	if err := applyGoalUpdate(goal, req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if err := db.DB.Save(goal).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "更新目标失败"})
	}

	return c.JSON(http.StatusOK, goal)
}

func (h *GoalHandler) Delete(c echo.Context) error {
	goal, err := findGoalByParam(c.Param("id"))
	if err != nil {
		return goalErrorResponse(c, err, "目标不存在", "无效的目标 ID", "获取目标失败")
	}

	goal.IsActive = false
	if err := db.DB.Save(goal).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "停用目标失败"})
	}

	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}

func (h *GoalHandler) UpsertRecord(c echo.Context) error {
	goal, err := findGoalByParam(c.Param("id"))
	if err != nil {
		return goalErrorResponse(c, err, "目标不存在", "无效的目标 ID", "获取目标失败")
	}

	recordDate, err := validateRecordDate(c.Param("date"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	var req UpsertGoalRecordRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "无效的请求数据"})
	}

	record, err := buildGoalRecord(goal, recordDate, req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	var existing models.GoalRecord
	result := db.DB.Where("goal_id = ? AND record_date = ?", goal.ID, recordDate).Take(&existing)
	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "保存打卡记录失败"})
	}

	if result.Error == gorm.ErrRecordNotFound {
		if err := db.DB.Create(&record).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "保存打卡记录失败"})
		}
		return c.JSON(http.StatusOK, toGoalRecordPayload(record))
	}

	existing.IsCompleted = record.IsCompleted
	existing.Quantity = record.Quantity
	if err := db.DB.Save(&existing).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "保存打卡记录失败"})
	}

	return c.JSON(http.StatusOK, toGoalRecordPayload(existing))
}

func (h *GoalHandler) DeleteRecord(c echo.Context) error {
	goal, err := findGoalByParam(c.Param("id"))
	if err != nil {
		return goalErrorResponse(c, err, "目标不存在", "无效的目标 ID", "获取目标失败")
	}

	_, err = validateRecordDate(c.Param("date"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	result := db.DB.Where("goal_id = ? AND record_date = ?", goal.ID, c.Param("date")).Delete(&models.GoalRecord{})
	if result.Error != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "删除打卡记录失败"})
	}

	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}

func (h *GoalHandler) Dashboard(c echo.Context) error {
	anchorDate, err := parseAnchorDate(c.QueryParam("date"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "日期格式无效，应为 YYYY-MM-DD"})
	}

	rangeKey := normalizeRange(c.QueryParam("range"))
	rangeStart, rangeEnd := rangeBounds(anchorDate, rangeKey)
	checkinDate, err := resolveCheckinDate(c.QueryParam("checkin_date"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	calendarMonth, monthStart, monthEnd, err := resolveCalendarMonth(c.QueryParam("month"), anchorDate)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "月份格式无效，应为 YYYY-MM"})
	}

	goals, err := listVisibleGoals()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "获取目标面板失败"})
	}
	inactiveGoals, err := listInactiveGoals()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "获取目标面板失败"})
	}

	recordsByGoal, err := loadRecordsByGoal(goals)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "获取目标面板失败"})
	}

	items := buildDashboardItems(goals, recordsByGoal, rangeStart, rangeEnd, anchorDate, checkinDate)
	calendarDays, dayDetails := buildCalendar(goals, recordsByGoal, monthStart, monthEnd)

	return c.JSON(http.StatusOK, GoalDashboardResponse{
		AnchorDate:     anchorDate.Format(goalDateLayout),
		Range:          rangeKey,
		RangeStartDate: rangeStart.Format(goalDateLayout),
		RangeEndDate:   rangeEnd.Format(goalDateLayout),
		CheckinDate:    checkinDate.Format(goalDateLayout),
		CalendarMonth:  calendarMonth,
		Goals:          items,
		InactiveGoals:  inactiveGoals,
		CalendarDays:   calendarDays,
		DayDetails:     dayDetails,
	})
}

func buildGoalFromCreate(req CreateGoalRequest) (models.Goal, error) {
	name := strings.TrimSpace(req.Name)
	goalType := strings.TrimSpace(req.GoalType)
	unit := strings.TrimSpace(req.Unit)
	if req.ReactivateID > 0 && goalType == "" {
		goalType = models.GoalTypeCheckbox
	}

	goal := models.Goal{
		Name:         name,
		GoalType:     goalType,
		Unit:         unit,
		AnnualTarget: normalizePositiveFloat(req.AnnualTarget),
		WeeklyTarget: normalizePositiveInt(req.WeeklyTarget),
		IsActive:     true,
	}

	if err := sanitizeAndValidateGoal(&goal); err != nil {
		return models.Goal{}, err
	}

	goal.SortOrder = nextGoalSortOrder()
	return goal, nil
}

func applyGoalUpdate(goal *models.Goal, req UpdateGoalRequest) error {
	if req.Name != nil {
		goal.Name = strings.TrimSpace(*req.Name)
	}
	if req.Unit != nil {
		goal.Unit = strings.TrimSpace(*req.Unit)
	}
	if req.AnnualTarget != nil {
		goal.AnnualTarget = normalizePositiveFloat(req.AnnualTarget)
	}
	if req.WeeklyTarget != nil {
		goal.WeeklyTarget = normalizePositiveInt(req.WeeklyTarget)
	}

	return sanitizeAndValidateGoal(goal)
}

func sanitizeAndValidateGoal(goal *models.Goal) error {
	if goal.Name == "" {
		return errors.New("目标名称不能为空")
	}

	switch goal.GoalType {
	case models.GoalTypeCheckbox:
		goal.Unit = ""
		goal.WeeklyTarget = nil
	case models.GoalTypeQuantity:
		if goal.Unit == "" {
			return errors.New("数值累计型目标需要填写单位")
		}
		goal.WeeklyTarget = nil
	case models.GoalTypeFrequency:
		if goal.WeeklyTarget == nil {
			return errors.New("频率型目标需要填写每周目标次数")
		}
		goal.Unit = ""
	default:
		return errors.New("无效的目标类型")
	}

	if goal.AnnualTarget != nil && *goal.AnnualTarget <= 0 {
		return errors.New("年度目标必须大于 0")
	}
	if goal.WeeklyTarget != nil && *goal.WeeklyTarget <= 0 {
		return errors.New("每周目标次数必须大于 0")
	}

	return nil
}

func buildGoalRecord(goal *models.Goal, recordDate string, req UpsertGoalRecordRequest) (models.GoalRecord, error) {
	record := models.GoalRecord{
		GoalID:      goal.ID,
		RecordDate:  recordDate,
		IsCompleted: true,
	}

	switch goal.GoalType {
	case models.GoalTypeCheckbox, models.GoalTypeFrequency:
		if req.Quantity != nil {
			return models.GoalRecord{}, errors.New("该目标不需要填写数值")
		}
	case models.GoalTypeQuantity:
		quantity := normalizePositiveFloat(req.Quantity)
		if quantity == nil {
			return models.GoalRecord{}, errors.New("数值累计型目标需要填写大于 0 的数量")
		}
		record.Quantity = quantity
	default:
		return models.GoalRecord{}, errors.New("无效的目标类型")
	}

	return record, nil
}

func listActiveGoals() ([]models.Goal, error) {
	var goals []models.Goal
	err := db.DB.Where("is_active = ?", true).Order("sort_order ASC").Order("id ASC").Find(&goals).Error
	return goals, err
}

func listVisibleGoals() ([]models.Goal, error) {
	return listActiveGoals()
}

func listInactiveGoals() ([]models.Goal, error) {
	var goals []models.Goal
	err := db.DB.Where("is_active = ?", false).Order("updated_at DESC").Order("id DESC").Find(&goals).Error
	return goals, err
}

func reactivateGoal(id uint, next models.Goal) (*models.Goal, error) {
	var goal models.Goal
	if err := db.DB.First(&goal, id).Error; err != nil {
		return nil, err
	}
	if goal.IsActive {
		return nil, errors.New("该目标已经启用")
	}

	goal.Name = next.Name
	goal.GoalType = next.GoalType
	goal.Unit = next.Unit
	goal.AnnualTarget = next.AnnualTarget
	goal.WeeklyTarget = next.WeeklyTarget
	goal.IsActive = true
	goal.SortOrder = nextGoalSortOrder()

	if err := sanitizeAndValidateGoal(&goal); err != nil {
		return nil, err
	}
	if err := db.DB.Save(&goal).Error; err != nil {
		return nil, err
	}

	return &goal, nil
}

func loadRecordsByGoal(goals []models.Goal) (map[uint][]models.GoalRecord, error) {
	if len(goals) == 0 {
		return map[uint][]models.GoalRecord{}, nil
	}

	ids := make([]uint, 0, len(goals))
	for _, goal := range goals {
		ids = append(ids, goal.ID)
	}

	var records []models.GoalRecord
	if err := db.DB.Where("goal_id IN ?", ids).Order("record_date ASC").Find(&records).Error; err != nil {
		return nil, err
	}

	result := make(map[uint][]models.GoalRecord, len(goals))
	for _, record := range records {
		result[record.GoalID] = append(result[record.GoalID], record)
	}

	return result, nil
}

func buildDashboardItems(goals []models.Goal, recordsByGoal map[uint][]models.GoalRecord, rangeStart, rangeEnd, anchorDate, checkinDate time.Time) []GoalDashboardItem {
	items := make([]GoalDashboardItem, 0, len(goals))
	weekStart, weekEnd := rangeBounds(anchorDate, "week")
	yearStart := time.Date(anchorDate.Year(), 1, 1, 0, 0, 0, 0, anchorDate.Location())
	yearEnd := time.Date(anchorDate.Year(), 12, 31, 0, 0, 0, 0, anchorDate.Location())
	checkinKey := checkinDate.Format(goalDateLayout)

	for _, goal := range goals {
		records := recordsByGoal[goal.ID]
		item := GoalDashboardItem{
			ID:           goal.ID,
			Name:         goal.Name,
			GoalType:     goal.GoalType,
			Unit:         goal.Unit,
			AnnualTarget: goal.AnnualTarget,
			WeeklyTarget: goal.WeeklyTarget,
		}

		for _, record := range records {
			recordTime, err := time.Parse(goalDateLayout, record.RecordDate)
			if err != nil {
				continue
			}

			if !recordTime.Before(rangeStart) && !recordTime.After(rangeEnd) {
				item.RangeCompletedCount++
				if record.Quantity != nil {
					item.RangeQuantityTotal += *record.Quantity
				}
			}

			if !recordTime.Before(yearStart) && !recordTime.After(yearEnd) {
				item.AnnualCompletedCount++
				if record.Quantity != nil {
					item.AnnualQuantityTotal += *record.Quantity
				}
			}

			if !recordTime.Before(weekStart) && !recordTime.After(weekEnd) {
				item.CurrentWeekCompletedCount++
			}

			if record.RecordDate == checkinKey {
				payload := toGoalRecordPayload(record)
				item.CheckinRecord = &payload
			}
		}

		switch goal.GoalType {
		case models.GoalTypeQuantity:
			if goal.AnnualTarget != nil {
				remaining := math.Max(*goal.AnnualTarget-item.AnnualQuantityTotal, 0)
				item.AnnualRemainingValue = floatPtr(roundToOneDecimal(remaining))
				percent := progressPercent(item.AnnualQuantityTotal, *goal.AnnualTarget)
				item.AnnualProgressPercent = &percent
			}
		case models.GoalTypeFrequency:
			if goal.WeeklyTarget != nil {
				percent := progressPercent(float64(item.CurrentWeekCompletedCount), float64(*goal.WeeklyTarget))
				item.CurrentWeekProgressPercent = &percent
			}
			if goal.AnnualTarget != nil {
				remaining := math.Max(*goal.AnnualTarget-float64(item.AnnualCompletedCount), 0)
				item.AnnualRemainingValue = floatPtr(roundToOneDecimal(remaining))
				percent := progressPercent(float64(item.AnnualCompletedCount), *goal.AnnualTarget)
				item.AnnualProgressPercent = &percent
			}
		case models.GoalTypeCheckbox:
			if goal.AnnualTarget != nil {
				remaining := math.Max(*goal.AnnualTarget-float64(item.AnnualCompletedCount), 0)
				item.AnnualRemainingValue = floatPtr(roundToOneDecimal(remaining))
				percent := progressPercent(float64(item.AnnualCompletedCount), *goal.AnnualTarget)
				item.AnnualProgressPercent = &percent
			}
		}

		item.RangeQuantityTotal = roundToOneDecimal(item.RangeQuantityTotal)
		item.AnnualQuantityTotal = roundToOneDecimal(item.AnnualQuantityTotal)
		items = append(items, item)
	}

	return items
}

func buildCalendar(goals []models.Goal, recordsByGoal map[uint][]models.GoalRecord, monthStart, monthEnd time.Time) ([]GoalCalendarDay, []GoalCalendarDayDetail) {
	totalGoals := len(goals)
	dayRecordMap := make(map[string]map[uint]models.GoalRecord)
	for goalID, records := range recordsByGoal {
		for _, record := range records {
			recordTime, err := time.Parse(goalDateLayout, record.RecordDate)
			if err != nil || recordTime.Before(monthStart) || recordTime.After(monthEnd) {
				continue
			}
			if _, exists := dayRecordMap[record.RecordDate]; !exists {
				dayRecordMap[record.RecordDate] = make(map[uint]models.GoalRecord)
			}
			dayRecordMap[record.RecordDate][goalID] = record
		}
	}

	days := make([]GoalCalendarDay, 0)
	details := make([]GoalCalendarDayDetail, 0)
	for current := monthStart; !current.After(monthEnd); current = current.AddDate(0, 0, 1) {
		dateStr := current.Format(goalDateLayout)
		recordMap := dayRecordMap[dateStr]
		completed := len(recordMap)
		intensity := 0
		if totalGoals > 0 && completed > 0 {
			ratio := float64(completed) / float64(totalGoals)
			switch {
			case ratio >= 1:
				intensity = 4
			case ratio >= 0.75:
				intensity = 3
			case ratio >= 0.4:
				intensity = 2
			default:
				intensity = 1
			}
		}

		days = append(days, GoalCalendarDay{
			Date:           dateStr,
			CompletedGoals: completed,
			TotalGoals:     totalGoals,
			Intensity:      intensity,
		})

		items := make([]GoalCalendarDayRecord, 0, len(goals))
		for _, goal := range goals {
			record, exists := recordMap[goal.ID]
			item := GoalCalendarDayRecord{
				GoalID:      goal.ID,
				Name:        goal.Name,
				GoalType:    goal.GoalType,
				Unit:        goal.Unit,
				IsCompleted: exists,
			}
			if exists {
				item.Quantity = record.Quantity
			}
			items = append(items, item)
		}

		details = append(details, GoalCalendarDayDetail{
			Date:           dateStr,
			CompletedGoals: completed,
			TotalGoals:     totalGoals,
			Items:          items,
		})
	}

	return days, details
}

func parseAnchorDate(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return today(), nil
	}
	return time.Parse(goalDateLayout, value)
}

func resolveCheckinDate(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return today().AddDate(0, 0, -1), nil
	}

	parsed, err := time.Parse(goalDateLayout, value)
	if err != nil {
		return time.Time{}, err
	}

	if !isEditableRecordDate(parsed) {
		return time.Time{}, errors.New("只能填写今天或昨天的打卡记录")
	}

	return parsed, nil
}

func resolveCalendarMonth(value string, anchorDate time.Time) (string, time.Time, time.Time, error) {
	if strings.TrimSpace(value) == "" {
		first := time.Date(anchorDate.Year(), anchorDate.Month(), 1, 0, 0, 0, 0, anchorDate.Location())
		return first.Format("2006-01"), first, first.AddDate(0, 1, -1), nil
	}

	parsed, err := time.Parse("2006-01", value)
	if err != nil {
		return "", time.Time{}, time.Time{}, err
	}
	first := time.Date(parsed.Year(), parsed.Month(), 1, 0, 0, 0, 0, parsed.Location())
	return first.Format("2006-01"), first, first.AddDate(0, 1, -1), nil
}

func rangeBounds(anchorDate time.Time, rangeKey string) (time.Time, time.Time) {
	switch rangeKey {
	case "week":
		weekday := int(anchorDate.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := time.Date(anchorDate.Year(), anchorDate.Month(), anchorDate.Day(), 0, 0, 0, 0, anchorDate.Location()).AddDate(0, 0, -(weekday - 1))
		return start, start.AddDate(0, 0, 6)
	case "quarter":
		quarterMonth := ((int(anchorDate.Month())-1)/3)*3 + 1
		start := time.Date(anchorDate.Year(), time.Month(quarterMonth), 1, 0, 0, 0, 0, anchorDate.Location())
		return start, start.AddDate(0, 3, -1)
	case "year":
		start := time.Date(anchorDate.Year(), 1, 1, 0, 0, 0, 0, anchorDate.Location())
		return start, time.Date(anchorDate.Year(), 12, 31, 0, 0, 0, 0, anchorDate.Location())
	case "all":
		var firstRecord models.GoalRecord
		var lastRecord models.GoalRecord
		if err := db.DB.Order("record_date ASC").First(&firstRecord).Error; err != nil {
			base := time.Date(anchorDate.Year(), anchorDate.Month(), 1, 0, 0, 0, 0, anchorDate.Location())
			return base, base.AddDate(0, 1, -1)
		}
		if err := db.DB.Order("record_date DESC").First(&lastRecord).Error; err != nil {
			base := time.Date(anchorDate.Year(), anchorDate.Month(), 1, 0, 0, 0, 0, anchorDate.Location())
			return base, base.AddDate(0, 1, -1)
		}
		start, _ := time.Parse(goalDateLayout, firstRecord.RecordDate)
		end, _ := time.Parse(goalDateLayout, lastRecord.RecordDate)
		return start, end
	default:
		start := time.Date(anchorDate.Year(), anchorDate.Month(), 1, 0, 0, 0, 0, anchorDate.Location())
		return start, start.AddDate(0, 1, -1)
	}
}

func normalizeRange(value string) string {
	switch strings.TrimSpace(value) {
	case "week", "month", "quarter", "year", "all":
		return strings.TrimSpace(value)
	default:
		return "month"
	}
}

func validateRecordDate(value string) (string, error) {
	parsed, err := time.Parse(goalDateLayout, value)
	if err != nil {
		return "", errors.New("日期格式无效，应为 YYYY-MM-DD")
	}
	if !isEditableRecordDate(parsed) {
		return "", errors.New("只能填写今天或昨天的打卡记录")
	}
	return parsed.Format(goalDateLayout), nil
}

func isEditableRecordDate(date time.Time) bool {
	t := today()
	y := t.AddDate(0, 0, -1)
	date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	return sameDay(date, t) || sameDay(date, y)
}

func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

func today() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

func nextGoalSortOrder() int {
	var maxSort int
	if err := db.DB.Model(&models.Goal{}).Select("COALESCE(MAX(sort_order), 0)").Scan(&maxSort).Error; err != nil {
		return 1
	}
	return maxSort + 1
}

func normalizePositiveFloat(value *float64) *float64 {
	if value == nil || *value <= 0 {
		return nil
	}
	rounded := roundToOneDecimal(*value)
	return &rounded
}

func normalizePositiveInt(value *int) *int {
	if value == nil || *value <= 0 {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func progressPercent(current, target float64) float64 {
	if target <= 0 {
		return 0
	}
	percent := (current / target) * 100
	if percent > 100 {
		percent = 100
	}
	return roundToOneDecimal(percent)
}

func roundToOneDecimal(value float64) float64 {
	return math.Round(value*10) / 10
}

func toGoalRecordPayload(record models.GoalRecord) GoalRecordPayload {
	return GoalRecordPayload{
		RecordDate:  record.RecordDate,
		IsCompleted: record.IsCompleted,
		Quantity:    record.Quantity,
	}
}

func findGoalByParam(idParam string) (*models.Goal, error) {
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		return nil, err
	}

	var goal models.Goal
	if err := db.DB.First(&goal, id).Error; err != nil {
		return nil, err
	}
	if !goal.IsActive {
		return nil, gorm.ErrRecordNotFound
	}
	return &goal, nil
}

func goalErrorResponse(c echo.Context, err error, notFoundMessage, badRequestMessage, internalMessage string) error {
	if _, parseErr := strconv.ParseUint(c.Param("id"), 10, 32); parseErr != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": badRequestMessage})
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return c.JSON(http.StatusNotFound, map[string]string{"error": notFoundMessage})
	}
	return c.JSON(http.StatusInternalServerError, map[string]string{"error": internalMessage})
}

func floatPtr(value float64) *float64 {
	return &value
}
