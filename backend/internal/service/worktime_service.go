package service

import (
	"database/sql"
	"fmt"
	"math"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jibiao-ai/deliverydesk/internal/model"
	"github.com/jibiao-ai/deliverydesk/internal/repository"
	"github.com/jibiao-ai/deliverydesk/pkg/logger"
)

// WorktimeService handles worktime statistics from Redmine
type WorktimeService struct{}

var (
	worktimeOnce    sync.Once
	worktimeService *WorktimeService
)

// GetWorktimeService returns singleton instance
func GetWorktimeService() *WorktimeService {
	worktimeOnce.Do(func() {
		worktimeService = &WorktimeService{}
	})
	return worktimeService
}

// Redmine connection config
const (
	redmineHost     = "172.16.7.83"
	redminePort     = 3306
	redmineUser     = "read1"
	redminePassword = "passw0rd"
	redmineDB       = "redmine"
)

// WorktimeEntry represents a single time entry from Redmine
type WorktimeEntry struct {
	ProjectName   string  `json:"project_name"`
	ProjectNo     string  `json:"project_no"`
	ContractParty string  `json:"contract_party"`
	EndUser       string  `json:"end_user"`
	Executor      string  `json:"executor"`
	Hours         float64 `json:"hours"`
	ManDayCost    float64 `json:"man_day_cost"`
	SpentOn       string  `json:"spent_on"`
	TaskName      string  `json:"task_name"`
	TaskSubject   string  `json:"task_subject"`
	WorkContent   string  `json:"work_content"`
}

// WorktimeUserStat represents aggregated worktime per user
type WorktimeUserStat struct {
	Name           string                `json:"name"`
	TotalHours     float64               `json:"total_hours"`
	TotalManDays   float64               `json:"total_man_days"`
	TotalCostDays  float64               `json:"total_cost_days"`
	ProjectDetails []WorktimeProjectStat `json:"project_details"`
}

// WorktimeProjectStat per-project stats within a user
type WorktimeProjectStat struct {
	ProjectName   string              `json:"project_name"`
	ProjectNo     string              `json:"project_no"`
	ContractParty string              `json:"contract_party"`
	EndUser       string              `json:"end_user"`
	TotalHours    float64             `json:"total_hours"`
	TotalManDays  float64             `json:"total_man_days"`
	TotalCostDays float64             `json:"total_cost_days"`
	MonthDetails  []WorktimeMonthStat `json:"month_details"`
}

// WorktimeMonthStat per-month stats within a project
type WorktimeMonthStat struct {
	Month      string              `json:"month"`
	TotalHours float64             `json:"total_hours"`
	ManDays    float64             `json:"man_days"`
	CostDays   float64             `json:"cost_days"`
	Tasks      []WorktimeTaskEntry `json:"tasks"`
}

// WorktimeTaskEntry represents task-level detail
type WorktimeTaskEntry struct {
	TaskName  string  `json:"task_name"`
	Hours     float64 `json:"hours"`
	ManDays   float64 `json:"man_days"`
	CostDays  float64 `json:"cost_days"`
	DateRange string  `json:"date_range"`
}

// WorktimeSummary is the top-level response
type WorktimeSummary struct {
	Period    string             `json:"period"`
	StartDate string            `json:"start_date"`
	EndDate   string            `json:"end_date"`
	Users     []WorktimeUserStat `json:"users"`
	Total     WorktimeTotals    `json:"total"`
}

// WorktimeTotals overall totals
type WorktimeTotals struct {
	TotalHours    float64 `json:"total_hours"`
	TotalManDays  float64 `json:"total_man_days"`
	TotalCostDays float64 `json:"total_cost_days"`
	UserCount     int     `json:"user_count"`
	ProjectCount  int     `json:"project_count"`
}

// GetDateRange calculates start/end dates based on period type
func GetDateRange(periodType string, refDate time.Time) (string, string) {
	switch periodType {
	case "month":
		start := time.Date(refDate.Year(), refDate.Month(), 1, 0, 0, 0, 0, refDate.Location())
		end := start.AddDate(0, 1, -1)
		return start.Format("2006-01-02"), end.Format("2006-01-02")
	case "quarter":
		q := (int(refDate.Month()) - 1) / 3
		startMonth := time.Month(q*3 + 1)
		start := time.Date(refDate.Year(), startMonth, 1, 0, 0, 0, 0, refDate.Location())
		end := start.AddDate(0, 3, -1)
		return start.Format("2006-01-02"), end.Format("2006-01-02")
	case "year":
		start := time.Date(refDate.Year(), 1, 1, 0, 0, 0, 0, refDate.Location())
		end := time.Date(refDate.Year(), 12, 31, 0, 0, 0, 0, refDate.Location())
		return start.Format("2006-01-02"), end.Format("2006-01-02")
	default:
		// default to current month
		start := time.Date(refDate.Year(), refDate.Month(), 1, 0, 0, 0, 0, refDate.Location())
		end := start.AddDate(0, 1, -1)
		return start.Format("2006-01-02"), end.Format("2006-01-02")
	}
}

// GetWorktimeStats fetches and aggregates worktime data for tracked users
func (s *WorktimeService) GetWorktimeStats(periodType string, startDate, endDate string) (*WorktimeSummary, error) {
	// Get tracked users
	users, err := s.ListUsers()
	if err != nil {
		return nil, fmt.Errorf("failed to get tracked users: %w", err)
	}

	if len(users) == 0 {
		return &WorktimeSummary{
			Period:    periodType,
			StartDate: startDate,
			EndDate:   endDate,
			Users:     []WorktimeUserStat{},
			Total:     WorktimeTotals{},
		}, nil
	}

	// Build user name list
	userNames := make([]string, len(users))
	for i, u := range users {
		userNames[i] = u.Name
	}

	// Query Redmine
	entries, err := s.queryRedmine(startDate, endDate, userNames)
	if err != nil {
		return nil, fmt.Errorf("failed to query redmine: %w", err)
	}

	// Aggregate data
	summary := s.aggregateEntries(entries, userNames, periodType, startDate, endDate)
	return summary, nil
}

// queryRedmine connects to Redmine MySQL and executes the worktime query
func (s *WorktimeService) queryRedmine(startDate, endDate string, userNames []string) ([]WorktimeEntry, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=true&loc=Local",
		redmineUser, redminePassword, redmineHost, redminePort, redmineDB)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to redmine DB: %w", err)
	}
	defer db.Close()

	db.SetConnMaxLifetime(30 * time.Second)
	db.SetMaxOpenConns(3)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping redmine DB: %w", err)
	}

	query := `
SELECT
	tt1.NAME AS project_name,
	tt1.projectNo AS project_no,
	tt1.contract_first_party AS contract_party,
	tt1.end_user,
	CONCAT(us2.lastname, us2.firstname) AS executor,
	te.hours,
	CASE WHEN te.hours >= 8 THEN 1 WHEN te.hours <= 4 THEN 0.5 ELSE 0.5 END AS man_day_cost,
	te.spent_on,
	t.NAME AS task_name,
	i.SUBJECT AS task_subject,
	te.comments AS work_content
FROM
	time_entries te
	LEFT JOIN issues i ON te.issue_id = i.id
	LEFT JOIN trackers t ON i.tracker_id = t.id
	LEFT JOIN (
		SELECT
			cv.customized_id,
			cv.VALUE AS executor
		FROM custom_values cv
		WHERE cv.custom_field_id = 3
	) tt ON tt.customized_id = te.id
	LEFT JOIN (
		SELECT
			p.id,
			p.NAME,
			t.projectNo,
			pes.contract_first_party,
			pes.end_user
		FROM projects p
		LEFT JOIN pms pm ON p.id = pm.project_id
		LEFT JOIN pm_stages ps ON (ps.pm_id = pm.id AND (stage_type = 'early' OR stage_type = 'start'))
		LEFT JOIN (
			SELECT pms1.stage_id, pms1.contract_first_party, pms1.end_user
			FROM pm_starts pms1
			WHERE pms1.created_at >= '2020-01-03 09:35:03'
			UNION ALL
			SELECT pme1.stage_id, pme1.contract_first_party, pme1.end_user
			FROM pm_earlies pme1
			WHERE pme1.created_at < '2020-01-03 09:35:03'
		) pes ON pes.stage_id = ps.id
		LEFT JOIN (
			SELECT c8.customized_id, c8.VALUE AS projectNo
			FROM custom_values c8
			WHERE c8.custom_field_id = 11
		) t ON p.id = t.customized_id
	) tt1 ON te.project_id = tt1.id
	LEFT JOIN users us2 ON tt.executor = us2.id
WHERE
	te.spent_on BETWEEN ? AND ?
ORDER BY te.spent_on`

	rows, err := db.Query(query, startDate+" 00:00:00", endDate+" 23:59:00")
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Build a set of tracked user names for filtering
	userSet := make(map[string]bool)
	for _, name := range userNames {
		userSet[name] = true
	}

	var entries []WorktimeEntry
	for rows.Next() {
		var e WorktimeEntry
		var projectName, projectNo, contractParty, endUser, executor, taskName, taskSubject, workContent sql.NullString
		var hours, manDayCost sql.NullFloat64
		var spentOn sql.NullTime

		err := rows.Scan(
			&projectName, &projectNo, &contractParty, &endUser, &executor,
			&hours, &manDayCost, &spentOn, &taskName, &taskSubject, &workContent,
		)
		if err != nil {
			logger.Log.Warnf("worktime scan error: %v", err)
			continue
		}

		executorName := ""
		if executor.Valid {
			executorName = executor.String
		}

		// Filter by tracked users
		if !userSet[executorName] {
			continue
		}

		e.ProjectName = nullStr(projectName)
		e.ProjectNo = nullStr(projectNo)
		e.ContractParty = nullStr(contractParty)
		e.EndUser = nullStr(endUser)
		e.Executor = executorName
		e.Hours = nullFloat(hours)
		e.ManDayCost = nullFloat(manDayCost)
		if spentOn.Valid {
			e.SpentOn = spentOn.Time.Format("2006-01-02")
		}
		e.TaskName = nullStr(taskName)
		e.TaskSubject = nullStr(taskSubject)
		e.WorkContent = nullStr(workContent)

		entries = append(entries, e)
	}

	logger.Log.Infof("Worktime: queried %d entries for %d users between %s and %s", len(entries), len(userNames), startDate, endDate)
	return entries, nil
}

func nullStr(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func nullFloat(nf sql.NullFloat64) float64 {
	if nf.Valid {
		return nf.Float64
	}
	return 0
}

// aggregateEntries processes raw entries into structured summary
func (s *WorktimeService) aggregateEntries(entries []WorktimeEntry, userNames []string, periodType, startDate, endDate string) *WorktimeSummary {
	// data[user][projectNo][month][taskName] = []entry
	type taskData struct {
		hours    float64
		costDays float64
		dates    []string
	}

	type monthData map[string]*taskData
	type projectData struct {
		info   WorktimeEntry // template for project info
		months map[string]monthData
	}

	userData := make(map[string]map[string]*projectData) // user -> projectNo -> data

	for _, e := range entries {
		if e.Executor == "" {
			continue
		}

		// Get month key
		monthKey := ""
		if len(e.SpentOn) >= 7 {
			t, err := time.Parse("2006-01-02", e.SpentOn)
			if err == nil {
				monthKey = fmt.Sprintf("%d/%d", t.Year(), int(t.Month()))
			}
		}
		if monthKey == "" {
			monthKey = "unknown"
		}

		projectKey := e.ProjectNo
		if projectKey == "" {
			projectKey = "no-project"
		}

		// Skip internal projects
		if projectKey == "ESS20210002" || projectKey == "ESS20210001" {
			continue
		}

		taskKey := e.TaskName
		if taskKey == "" {
			taskKey = "其他"
		}

		// Initialize nested maps
		if userData[e.Executor] == nil {
			userData[e.Executor] = make(map[string]*projectData)
		}
		if userData[e.Executor][projectKey] == nil {
			userData[e.Executor][projectKey] = &projectData{
				info:   e,
				months: make(map[string]monthData),
			}
		}
		if userData[e.Executor][projectKey].months[monthKey] == nil {
			userData[e.Executor][projectKey].months[monthKey] = make(monthData)
		}
		if userData[e.Executor][projectKey].months[monthKey][taskKey] == nil {
			userData[e.Executor][projectKey].months[monthKey][taskKey] = &taskData{}
		}

		td := userData[e.Executor][projectKey].months[monthKey][taskKey]
		td.hours += e.Hours
		td.costDays += e.ManDayCost
		td.dates = append(td.dates, e.SpentOn)
	}

	// Build summary
	var userStats []WorktimeUserStat
	var grandTotalHours, grandTotalManDays, grandTotalCostDays float64
	projectSet := make(map[string]bool)

	for _, userName := range userNames {
		projects, exists := userData[userName]
		if !exists {
			// User has no entries in this period, still include with zero
			userStats = append(userStats, WorktimeUserStat{
				Name:           userName,
				TotalHours:     0,
				TotalManDays:   0,
				TotalCostDays:  0,
				ProjectDetails: []WorktimeProjectStat{},
			})
			continue
		}

		var userTotalHours, userTotalManDays, userTotalCostDays float64
		var projectStats []WorktimeProjectStat

		for projNo, pd := range projects {
			projectSet[projNo] = true
			var projHours, projManDays, projCostDays float64
			var monthStats []WorktimeMonthStat

			for month, tasks := range pd.months {
				var mHours, mManDays, mCostDays float64
				var taskEntries []WorktimeTaskEntry

				for taskName, td := range tasks {
					manDays := math.Round(td.hours / 8)
					mHours += td.hours
					mManDays += manDays
					mCostDays += td.costDays

					dateRange := ""
					if len(td.dates) > 0 {
						dateRange = td.dates[0]
						if len(td.dates) > 1 {
							dateRange = td.dates[0] + " ~ " + td.dates[len(td.dates)-1]
						}
					}

					taskEntries = append(taskEntries, WorktimeTaskEntry{
						TaskName:  taskName,
						Hours:     td.hours,
						ManDays:   manDays,
						CostDays:  td.costDays,
						DateRange: dateRange,
					})
				}

				monthStats = append(monthStats, WorktimeMonthStat{
					Month:      month,
					TotalHours: mHours,
					ManDays:    mManDays,
					CostDays:   mCostDays,
					Tasks:      taskEntries,
				})

				projHours += mHours
				projManDays += mManDays
				projCostDays += mCostDays
			}

			projectStats = append(projectStats, WorktimeProjectStat{
				ProjectName:   pd.info.ProjectName,
				ProjectNo:     projNo,
				ContractParty: pd.info.ContractParty,
				EndUser:       pd.info.EndUser,
				TotalHours:    projHours,
				TotalManDays:  projManDays,
				TotalCostDays: projCostDays,
				MonthDetails:  monthStats,
			})

			userTotalHours += projHours
			userTotalManDays += projManDays
			userTotalCostDays += projCostDays
		}

		userStats = append(userStats, WorktimeUserStat{
			Name:           userName,
			TotalHours:     userTotalHours,
			TotalManDays:   userTotalManDays,
			TotalCostDays:  userTotalCostDays,
			ProjectDetails: projectStats,
		})

		grandTotalHours += userTotalHours
		grandTotalManDays += userTotalManDays
		grandTotalCostDays += userTotalCostDays
	}

	return &WorktimeSummary{
		Period:    periodType,
		StartDate: startDate,
		EndDate:   endDate,
		Users:     userStats,
		Total: WorktimeTotals{
			TotalHours:    grandTotalHours,
			TotalManDays:  grandTotalManDays,
			TotalCostDays: grandTotalCostDays,
			UserCount:     len(userNames),
			ProjectCount:  len(projectSet),
		},
	}
}

// ListUsers returns all tracked worktime users
func (s *WorktimeService) ListUsers() ([]model.WorktimeUser, error) {
	var users []model.WorktimeUser
	if err := repository.DB.Order("name ASC").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// AddUser adds a new tracked user
func (s *WorktimeService) AddUser(name string, addedBy uint) (*model.WorktimeUser, error) {
	user := &model.WorktimeUser{
		Name:    name,
		AddedBy: addedBy,
	}
	if err := repository.DB.Create(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

// RemoveUser removes a tracked user
func (s *WorktimeService) RemoveUser(id uint) error {
	return repository.DB.Delete(&model.WorktimeUser{}, id).Error
}

// BatchAddUsers adds multiple users at once
func (s *WorktimeService) BatchAddUsers(names []string, addedBy uint) ([]model.WorktimeUser, error) {
	var created []model.WorktimeUser
	for _, name := range names {
		if name == "" {
			continue
		}
		user := model.WorktimeUser{Name: name, AddedBy: addedBy}
		// Use FirstOrCreate to avoid duplicates
		result := repository.DB.Where("name = ?", name).FirstOrCreate(&user)
		if result.Error != nil {
			logger.Log.Warnf("Failed to add worktime user %s: %v", name, result.Error)
			continue
		}
		created = append(created, user)
	}
	return created, nil
}
