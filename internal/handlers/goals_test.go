package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"e4-api/internal/db"
	"e4-api/internal/models"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestGoalHandler(t *testing.T) (*GoalHandler, *echo.Echo) {
	t.Helper()

	dsn := fmt.Sprintf("file:goal-test-%d?mode=memory&cache=shared", time.Now().UnixNano())
	database, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&models.Goal{}, &models.GoalRecord{}))

	db.DB = database

	return NewGoalHandler(), echo.New()
}

func seedGoals(t *testing.T, goals []models.Goal) {
	t.Helper()
	for _, goal := range goals {
		require.NoError(t, db.DB.Create(&goal).Error)
	}
}

func seedGoalRecords(t *testing.T, records []models.GoalRecord) {
	t.Helper()
	for _, record := range records {
		require.NoError(t, db.DB.Create(&record).Error)
	}
}

func seedInactiveGoal(t *testing.T, goal models.Goal) {
	t.Helper()
	goal.IsActive = true
	require.NoError(t, db.DB.Create(&goal).Error)
	require.NoError(t, db.DB.Model(&models.Goal{}).Where("id = ?", goal.ID).Update("is_active", false).Error)
}

func TestGoalCreateRejectsInvalidQuantityGoal(t *testing.T) {
	handler, e := setupTestGoalHandler(t)
	body := bytes.NewBufferString(`{"name":"跑步","goal_type":"quantity"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/goals", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Create(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.JSONEq(t, `{"error":"数值累计型目标需要填写单位"}`, rec.Body.String())
}

func TestGoalUpsertRecordRejectsOlderDate(t *testing.T) {
	handler, e := setupTestGoalHandler(t)
	seedGoals(t, []models.Goal{{Name: "冥想", GoalType: models.GoalTypeCheckbox, IsActive: true, SortOrder: 1}})

	olderDate := time.Now().AddDate(0, 0, -2).Format(goalDateLayout)
	req := httptest.NewRequest(http.MethodPut, "/api/goals/1/records/"+olderDate, bytes.NewBufferString(`{}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id", "date")
	c.SetParamValues("1", olderDate)

	err := handler.UpsertRecord(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.JSONEq(t, `{"error":"只能填写今天或昨天的打卡记录"}`, rec.Body.String())
}

func TestGoalUpsertRecordStoresQuantity(t *testing.T) {
	handler, e := setupTestGoalHandler(t)
	annualTarget := 500.0
	seedGoals(t, []models.Goal{{Name: "跑步", GoalType: models.GoalTypeQuantity, Unit: "km", AnnualTarget: &annualTarget, IsActive: true, SortOrder: 1}})

	date := time.Now().AddDate(0, 0, -1).Format(goalDateLayout)
	req := httptest.NewRequest(http.MethodPut, "/api/goals/1/records/"+date, bytes.NewBufferString(`{"quantity":5.2}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id", "date")
	c.SetParamValues("1", date)

	err := handler.UpsertRecord(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var payload GoalRecordPayload
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.NotNil(t, payload.Quantity)
	assert.Equal(t, 5.2, *payload.Quantity)
	assert.Equal(t, date, payload.RecordDate)
}

func TestGoalDashboardBuildsAggregates(t *testing.T) {
	handler, e := setupTestGoalHandler(t)
	annualRun := 500.0
	weeklyRun := 3
	seedGoals(t, []models.Goal{
		{Name: "冥想", GoalType: models.GoalTypeCheckbox, IsActive: true, SortOrder: 1},
		{Name: "跑步", GoalType: models.GoalTypeQuantity, Unit: "km", AnnualTarget: &annualRun, IsActive: true, SortOrder: 2},
		{Name: "打球", GoalType: models.GoalTypeFrequency, WeeklyTarget: &weeklyRun, IsActive: true, SortOrder: 3},
	})

	anchor := today()
	yesterday := anchor.AddDate(0, 0, -1)
	monthStart := time.Date(anchor.Year(), anchor.Month(), 1, 0, 0, 0, 0, anchor.Location())
	seedGoalRecords(t, []models.GoalRecord{
		{GoalID: 1, RecordDate: yesterday.Format(goalDateLayout), IsCompleted: true},
		{GoalID: 1, RecordDate: monthStart.Format(goalDateLayout), IsCompleted: true},
		{GoalID: 2, RecordDate: yesterday.Format(goalDateLayout), IsCompleted: true, Quantity: floatPtr(5)},
		{GoalID: 2, RecordDate: monthStart.Format(goalDateLayout), IsCompleted: true, Quantity: floatPtr(3)},
		{GoalID: 3, RecordDate: yesterday.Format(goalDateLayout), IsCompleted: true},
	})

	url := fmt.Sprintf("/api/goals/dashboard?range=month&date=%s&checkin_date=%s&month=%s", anchor.Format(goalDateLayout), yesterday.Format(goalDateLayout), anchor.Format("2006-01"))
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Dashboard(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var response GoalDashboardResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response.Goals, 3)
	assert.Equal(t, yesterday.Format(goalDateLayout), response.CheckinDate)
	assert.NotEmpty(t, response.CalendarDays)

	runGoal := response.Goals[1]
	assert.Equal(t, "跑步", runGoal.Name)
	assert.Equal(t, 8.0, runGoal.RangeQuantityTotal)
	assert.Equal(t, 8.0, runGoal.AnnualQuantityTotal)
	require.NotNil(t, runGoal.AnnualRemainingValue)
	assert.Equal(t, 492.0, *runGoal.AnnualRemainingValue)

	frequencyGoal := response.Goals[2]
	require.NotNil(t, frequencyGoal.CurrentWeekProgressPercent)
	assert.GreaterOrEqual(t, *frequencyGoal.CurrentWeekProgressPercent, 0.0)
	assert.LessOrEqual(t, *frequencyGoal.CurrentWeekProgressPercent, 100.0)
	assert.NotNil(t, frequencyGoal.CheckinRecord)
	assert.Equal(t, yesterday.Format(goalDateLayout), frequencyGoal.CheckinRecord.RecordDate)
}

func TestGoalCreateAndListFlow(t *testing.T) {
	handler, e := setupTestGoalHandler(t)
	body := bytes.NewBufferString(`{"name":"读书","goal_type":"quantity","unit":"本","annual_target":24}`)
	req := httptest.NewRequest(http.MethodPost, "/api/goals", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Create(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, rec.Code)

	listReq := httptest.NewRequest(http.MethodGet, "/api/goals", nil)
	listRec := httptest.NewRecorder()
	listCtx := e.NewContext(listReq, listRec)

	err = handler.List(listCtx)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, listRec.Code)

	var payload struct {
		Goals []models.Goal `json:"goals"`
	}
	require.NoError(t, json.Unmarshal(listRec.Body.Bytes(), &payload))
	require.Len(t, payload.Goals, 1)
	assert.Equal(t, "读书", payload.Goals[0].Name)
	assert.Equal(t, models.GoalTypeQuantity, payload.Goals[0].GoalType)
}

func TestGoalUpdateSupportsEditingAnnualTarget(t *testing.T) {
	handler, e := setupTestGoalHandler(t)
	annualTarget := 12.0
	seedGoals(t, []models.Goal{{Name: "看电影", GoalType: models.GoalTypeCheckbox, AnnualTarget: &annualTarget, IsActive: true, SortOrder: 1}})

	body := bytes.NewBufferString(`{"name":"电影清单","annual_target":18}`)
	req := httptest.NewRequest(http.MethodPut, "/api/goals/1", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := handler.Update(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var updated models.Goal
	require.NoError(t, db.DB.First(&updated, 1).Error)
	assert.Equal(t, "电影清单", updated.Name)
	require.NotNil(t, updated.AnnualTarget)
	assert.Equal(t, 18.0, *updated.AnnualTarget)
}

func TestGoalDeleteMarksGoalInactive(t *testing.T) {
	handler, e := setupTestGoalHandler(t)
	seedGoals(t, []models.Goal{{Name: "早起拉伸", GoalType: models.GoalTypeCheckbox, IsActive: true, SortOrder: 1}})

	req := httptest.NewRequest(http.MethodDelete, "/api/goals/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := handler.Delete(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var goal models.Goal
	require.NoError(t, db.DB.First(&goal, 1).Error)
	assert.False(t, goal.IsActive)
}

func TestGoalDeleteRecordRemovesExistingRecord(t *testing.T) {
	handler, e := setupTestGoalHandler(t)
	seedGoals(t, []models.Goal{{Name: "冥想", GoalType: models.GoalTypeCheckbox, IsActive: true, SortOrder: 1}})
	recordDate := time.Now().AddDate(0, 0, -1).Format(goalDateLayout)
	seedGoalRecords(t, []models.GoalRecord{{GoalID: 1, RecordDate: recordDate, IsCompleted: true}})

	req := httptest.NewRequest(http.MethodDelete, "/api/goals/1/records/"+recordDate, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id", "date")
	c.SetParamValues("1", recordDate)

	err := handler.DeleteRecord(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var count int64
	require.NoError(t, db.DB.Model(&models.GoalRecord{}).Where("goal_id = ? AND record_date = ?", 1, recordDate).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func TestGoalUpsertRecordRejectsQuantityForCheckboxGoal(t *testing.T) {
	handler, e := setupTestGoalHandler(t)
	seedGoals(t, []models.Goal{{Name: "冥想", GoalType: models.GoalTypeCheckbox, IsActive: true, SortOrder: 1}})
	recordDate := time.Now().AddDate(0, 0, -1).Format(goalDateLayout)

	req := httptest.NewRequest(http.MethodPut, "/api/goals/1/records/"+recordDate, bytes.NewBufferString(`{"quantity":2}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id", "date")
	c.SetParamValues("1", recordDate)

	err := handler.UpsertRecord(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.JSONEq(t, `{"error":"该目标不需要填写数值"}`, rec.Body.String())
}

func TestGoalCreateCanReactivateInactiveGoal(t *testing.T) {
	handler, e := setupTestGoalHandler(t)
	seedInactiveGoal(t, models.Goal{ID: 1, Name: "旧目标", GoalType: models.GoalTypeCheckbox, SortOrder: 1})

	body := bytes.NewBufferString(`{"reactivate_id":1,"name":"冥想训练","goal_type":"checkbox","annual_target":180}`)
	req := httptest.NewRequest(http.MethodPost, "/api/goals", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Create(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, rec.Code)

	var goal models.Goal
	require.NoError(t, db.DB.First(&goal, 1).Error)
	assert.True(t, goal.IsActive)
	assert.Equal(t, "冥想训练", goal.Name)
	require.NotNil(t, goal.AnnualTarget)
	assert.Equal(t, 180.0, *goal.AnnualTarget)
}

func TestGoalDashboardIncludesInactiveGoals(t *testing.T) {
	handler, e := setupTestGoalHandler(t)
	seedGoals(t, []models.Goal{{ID: 1, Name: "活跃目标", GoalType: models.GoalTypeCheckbox, IsActive: true, SortOrder: 1}})
	seedInactiveGoal(t, models.Goal{ID: 2, Name: "已停用目标", GoalType: models.GoalTypeCheckbox, SortOrder: 2})

	req := httptest.NewRequest(http.MethodGet, "/api/goals/dashboard", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Dashboard(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var response GoalDashboardResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response.Goals, 1)
	require.Len(t, response.InactiveGoals, 1)
	assert.Equal(t, "已停用目标", response.InactiveGoals[0].Name)
}
