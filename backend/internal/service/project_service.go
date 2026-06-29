package service

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jibiao-ai/deliverydesk/internal/model"
	"github.com/jibiao-ai/deliverydesk/internal/repository"
	"github.com/jibiao-ai/deliverydesk/pkg/logger"
	"gorm.io/gorm"
)

// ProjectService handles project management statistics from Redmine
type ProjectService struct{}

var (
	projectOnce    sync.Once
	projectService *ProjectService
)

// GetProjectService returns singleton instance
func GetProjectService() *ProjectService {
	projectOnce.Do(func() {
		projectService = &ProjectService{}
	})
	return projectService
}

// ProjectStats aggregates project statistics for the frontend
type ProjectStats struct {
	TotalProjects     int                    `json:"total_projects"`
	PeriodNew         int                    `json:"period_new"`           // new projects in selected period
	WeekNew           int                    `json:"week_new"`            // new projects in past 7 days
	MonthlyTrend      []MonthlyCount         `json:"monthly_trend"`       // line chart data
	RegionDistribution []NameCount           `json:"region_distribution"` // donut chart data
	TypeDistribution   []NameCount           `json:"type_distribution"`   // donut chart data
	StatusDistribution []NameCount           `json:"status_distribution"` // bar chart data
	Top5Regions       []NameCount            `json:"top5_regions"`        // TOP5 regions
	Top5Managers      []NameCount            `json:"top5_managers"`       // TOP5 project managers
	Top5Customers     []NameCount            `json:"top5_customers"`      // TOP5 customers
	YoYComparison     []YoYData             `json:"yoy_comparison"`      // year-over-year comparison
	WeeklyProjects    []DailyCount           `json:"weekly_projects"`     // past week daily creation
	LastSyncTime      string                 `json:"last_sync_time"`
}

// MonthlyCount represents monthly project creation count
type MonthlyCount struct {
	Month string `json:"month"` // "2024-01" format
	Count int    `json:"count"`
}

// DailyCount represents daily project creation count
type DailyCount struct {
	Date  string `json:"date"` // "2024-01-15" format
	Count int    `json:"count"`
}

// NameCount represents name-count pair for charts
type NameCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// YoYData represents year-over-year comparison data
type YoYData struct {
	Year  int `json:"year"`
	Count int `json:"count"`
}

// redmineProjectSQL is the full query from the reference script
const redmineProjectSQL = `
SELECT
    tt1.NAME AS project_name,
    tt1.projectNo AS project_no,
    tt1.sales_director,
    tt1.sale_vp,
    tt1.sale_user,
    tt1.pre_sale_user,
    tt1.sale_order_number,
    tt1.contract_number,
    tt1.contract_first_party,
    tt1.end_user,
    tt1.contract_date,
    tt1.created_on AS project_start_date,
    IF(tt1.created_at >= '2020-01-03 09:35:03', tt1.pmi1_start_date, tt1.pmex1_start_date) AS delivery_start_date,
    IF(tt1.created_at >= '2020-01-03 09:35:03', tt1.pmi1_end_date, tt1.pmex1_end_date) AS delivery_end_date,
    IF(tt1.created_at >= '2020-01-03 09:35:03', tt1.pma1_start_date, tt1.pmcl1_start_date) AS accept_start_date,
    IF(tt1.created_at >= '2020-01-03 09:35:03', tt1.pma1_end_date, tt1.pmcl1_end_date) AS accept_end_date,
    tt1.projectCktime AS expect_accept_date,
    tt1.projectCk AS project_acceptance,
    CONCAT(us.lastname, us.firstname) AS project_manager,
    CASE
        WHEN tt1.STATUS = "1" THEN "正常"
        WHEN tt1.STATUS = "0" THEN "异常"
        ELSE ""
    END AS status,
    tt1.are AS region,
    tt1.pro AS province,
    tt1.businessNo AS business_no,
    tt1.payType AS delivery_type,
    CASE
        WHEN tt1.keyPoint = "0" THEN "否"
        WHEN tt1.keyPoint = "1" THEN "是"
        ELSE ""
    END AS is_key_project,
    tt1.customType AS customer_type,
    tt1.project_type,
    tt1.current_state AS project_status
FROM
    (
    SELECT
        p.id,
        p.NAME,
        t.projectNo,
        p.created_on,
        pes.sales_director,
        pes.created_at,
        pes.sale_vp,
        pes.sale_user,
        pes.pre_sale_user,
        pes.sale_order_number,
        pes.contract_number,
        pes.contract_first_party,
        pes.end_user,
        pes.contract_date,
        pes.pmi1_start_date,
        pes.pmi1_end_date,
        pes.pma1_start_date,
        pes.pma1_end_date,
        pes.pmex1_start_date,
        pes.pmex1_end_date,
        pes.pmcl1_start_date,
        pes.pmcl1_end_date,
        t.proowner,
        p.STATUS,
        t.are,
        t.pro,
        t.businessNo,
        t.payType,
        t.keyPoint,
        t.customType,
        t.projectCk,
        t.projectCktime,
        pm.project_type,
        pm.current_state
    FROM
        projects p
        LEFT JOIN pms pm ON p.id = pm.project_id
        LEFT JOIN pm_stages ps ON (
            ps.pm_id = pm.id
        AND ( stage_type = 'early' OR stage_type = 'start' ))
        LEFT JOIN (
        SELECT
            pms1.stage_id,
            pms1.created_at,
            pms1.sale_user,
            pms1.pre_sale_user,
            pms1.has_POC,
            pms1.sale_vp,
            pms1.contract_number,
            pms1.contract_first_party,
            pms1.sale_order_number,
            pms1.end_user,
            pms1.sales_director,
            pms1.contract_date,
            pmi1.start_date AS pmi1_start_date,
            pmi1.end_date AS pmi1_end_date,
            pma1.start_date AS pma1_start_date,
            pma1.end_date AS pma1_end_date,
            "" AS pmex1_start_date,
            "" AS pmex1_end_date,
            "" AS pmcl1_start_date,
            "" AS pmcl1_end_date
        FROM
            pm_starts pms1
            LEFT JOIN pm_implementations pmi1 on pms1.stage_id = pmi1.stage_id - 1
            LEFT JOIN pm_acceptances pma1 on pms1.stage_id = pma1.stage_id - 2
        WHERE
            pms1.created_at >= '2020-01-03 09:35:03' UNION ALL
        SELECT
            pme1.stage_id,
            pme1.created_at,
            pme1.sale_user,
            pme1.pre_sale_user,
            pme1.has_POC,
            pme1.sale_vp,
            pme1.contract_number,
            pme1.contract_first_party,
            pme1.sale_order_number,
            pme1.end_user,
            pme1.sales_director,
            pme1.contract_date,
            "" AS pmi1_start_date,
            "" AS pmi1_end_date,
            "" AS pma1_start_date,
            "" AS pma1_end_date,
            pmex1.start_date AS pmex1_start_date,
            pmex1.end_date AS pmex1_end_date,
            pmcl1.start_date AS pmcl1_start_date,
            pmcl1.end_date AS pmcl1_end_date
        FROM
            pm_earlies pme1
            LEFT JOIN pm_executes pmex1 ON pme1.stage_id = pmex1.stage_id - 1
            LEFT JOIN pm_closes pmcl1 ON pme1.stage_id = pmcl1.stage_id - 2
        WHERE
            pme1.created_at < '2020-01-03 09:35:03'
        ) pes ON pes.stage_id = ps.id
        LEFT JOIN (
        SELECT
            c1.customized_id,
            c1.VALUE AS pro,
            c2.VALUE AS are,
            c3.VALUE AS businessNo,
            c4.VALUE AS payType,
            c5.VALUE AS contractNo,
            c6.VALUE AS customType,
            c7.VALUE AS keyPoint,
            c8.VALUE AS projectNo,
            c9.VALUE AS proowner,
            c10.VALUE AS projectCk,
            c11.VALUE AS projectCktime
        FROM
            custom_values c1
            LEFT JOIN custom_values c2 ON ( c1.customized_id = c2.customized_id AND c2.custom_field_id = 38 )
            LEFT JOIN custom_values c3 ON ( c1.customized_id = c3.customized_id AND c3.custom_field_id = 61 )
            LEFT JOIN custom_values c4 ON ( c1.customized_id = c4.customized_id AND c4.custom_field_id = 53 )
            LEFT JOIN custom_values c5 ON ( c1.customized_id = c5.customized_id AND c5.custom_field_id = 63 )
            LEFT JOIN custom_values c6 ON ( c1.customized_id = c6.customized_id AND c6.custom_field_id = 66 )
            LEFT JOIN custom_values c7 ON ( c1.customized_id = c7.customized_id AND c7.custom_field_id = 55 )
            LEFT JOIN custom_values c8 ON ( c1.customized_id = c8.customized_id AND c8.custom_field_id = 11 )
            LEFT JOIN custom_values c9 ON ( c1.customized_id = c9.customized_id AND c9.custom_field_id = 17 )
            LEFT JOIN custom_values c10 ON ( c1.customized_id = c10.customized_id AND c10.custom_field_id = 68 )
            LEFT JOIN custom_values c11 ON ( c1.customized_id = c11.customized_id AND c11.custom_field_id = 69 )
        WHERE
            c1.custom_field_id = 58
        ) t ON p.id = t.customized_id
    ) tt1
    LEFT JOIN users us ON tt1.proowner = us.id
WHERE
    tt1.NAME NOT LIKE '0%'
    AND tt1.NAME NOT LIKE '客户成功部%'
    AND tt1.NAME NOT LIKE '%POC%'
`

// SyncFromRedmine fetches all project data from Redmine and stores in local DB
func (s *ProjectService) SyncFromRedmine() error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=30s",
		redmineUser, redminePassword, redmineHost, redminePort, redmineDB)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("connect to redmine failed: %w", err)
	}
	defer db.Close()

	db.SetConnMaxLifetime(2 * time.Minute)
	db.SetMaxOpenConns(5)

	rows, err := db.Query(redmineProjectSQL)
	if err != nil {
		return fmt.Errorf("query redmine projects failed: %w", err)
	}
	defer rows.Close()

	var projects []model.ProjectInfo
	now := time.Now()

	for rows.Next() {
		var p model.ProjectInfo
		var projectName, projectNo, salesDirector, saleVP, saleUser, preSaleUser sql.NullString
		var saleOrderNumber, contractNumber, contractParty, endUser, contractDate sql.NullString
		var projectStartDate, deliveryStartDate, deliveryEndDate sql.NullString
		var acceptStartDate, acceptEndDate, expectAcceptDate, projectAcceptance sql.NullString
		var projectManager, status, region, province, businessNo sql.NullString
		var deliveryType, isKeyProject, customerType, projectType, projectStatus sql.NullString

		err := rows.Scan(
			&projectName, &projectNo, &salesDirector, &saleVP, &saleUser, &preSaleUser,
			&saleOrderNumber, &contractNumber, &contractParty, &endUser, &contractDate,
			&projectStartDate, &deliveryStartDate, &deliveryEndDate,
			&acceptStartDate, &acceptEndDate, &expectAcceptDate, &projectAcceptance,
			&projectManager, &status, &region, &province, &businessNo,
			&deliveryType, &isKeyProject, &customerType, &projectType, &projectStatus,
		)
		if err != nil {
			logger.Log.Warnf("scan project row error: %v", err)
			continue
		}

		p.ProjectName = nullStrP(projectName)
		p.ProjectNo = nullStrP(projectNo)
		p.SalesDirector = nullStrP(salesDirector)
		p.SaleVP = nullStrP(saleVP)
		p.SaleUser = nullStrP(saleUser)
		p.PreSaleUser = nullStrP(preSaleUser)
		p.SaleOrderNumber = nullStrP(saleOrderNumber)
		p.ContractNumber = nullStrP(contractNumber)
		p.ContractParty = nullStrP(contractParty)
		p.EndUser = nullStrP(endUser)
		p.ContractDate = nullStrP(contractDate)
		p.ProjectStartDate = nullStrP(projectStartDate)
		p.DeliveryStartDate = nullStrP(deliveryStartDate)
		p.DeliveryEndDate = nullStrP(deliveryEndDate)
		p.AcceptStartDate = nullStrP(acceptStartDate)
		p.AcceptEndDate = nullStrP(acceptEndDate)
		p.ExpectAcceptDate = nullStrP(expectAcceptDate)
		p.ProjectAcceptance = nullStrP(projectAcceptance)
		p.ProjectManager = nullStrP(projectManager)
		p.Status = nullStrP(status)
		p.Region = nullStrP(region)
		p.Province = nullStrP(province)
		p.BusinessNo = nullStrP(businessNo)
		p.DeliveryType = nullStrP(deliveryType)
		p.IsKeyProject = nullStrP(isKeyProject)
		p.CustomerType = nullStrP(customerType)
		p.ProjectType = nullStrP(projectType)
		p.ProjectStatus = nullStrP(projectStatus)
		p.SyncedAt = now

		projects = append(projects, p)
	}

	if len(projects) == 0 {
		return fmt.Errorf("no projects fetched from Redmine")
	}

	logger.Log.Infof("Fetched %d projects from Redmine, syncing to local DB...", len(projects))

	// Clear existing data and bulk insert (transactional)
	tx := repository.DB.Begin()
	if err := tx.Exec("DELETE FROM project_infos").Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("clear project_infos failed: %w", err)
	}

	// Batch insert in chunks of 100
	batchSize := 100
	for i := 0; i < len(projects); i += batchSize {
		end := i + batchSize
		if end > len(projects) {
			end = len(projects)
		}
		if err := tx.Create(projects[i:end]).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("batch insert projects failed: %w", err)
		}
	}

	tx.Commit()
	logger.Log.Infof("Successfully synced %d projects to local DB", len(projects))
	return nil
}

// GetStats returns aggregated project statistics based on period filter
func (s *ProjectService) GetStats(periodType string, startDate string, endDate string) (*ProjectStats, error) {
	stats := &ProjectStats{}
	db := repository.DB

	// Total projects count
	var total int64
	db.Model(&model.ProjectInfo{}).Count(&total)
	stats.TotalProjects = int(total)

	// Get last sync time
	var lastProject model.ProjectInfo
	if err := db.Order("synced_at DESC").First(&lastProject).Error; err == nil {
		stats.LastSyncTime = lastProject.SyncedAt.Format("2006-01-02 15:04:05")
	}

	// Period filter - count new projects in selected period
	if startDate != "" && endDate != "" {
		var periodCount int64
		db.Model(&model.ProjectInfo{}).
			Where("project_start_date >= ? AND project_start_date <= ?", startDate, endDate+" 23:59:59").
			Count(&periodCount)
		stats.PeriodNew = int(periodCount)
	}

	// Past 7 days new projects
	weekAgo := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	var weekCount int64
	db.Model(&model.ProjectInfo{}).
		Where("project_start_date >= ?", weekAgo).
		Count(&weekCount)
	stats.WeekNew = int(weekCount)

	// Monthly trend (last 12 months)
	stats.MonthlyTrend = s.getMonthlyTrend(db)

	// Region distribution
	stats.RegionDistribution = s.getDistribution(db, "region")

	// Project type distribution
	stats.TypeDistribution = s.getDistribution(db, "project_type")

	// Project status distribution
	stats.StatusDistribution = s.getDistribution(db, "project_status")

	// TOP5 regions
	stats.Top5Regions = s.getTop5(db, "region")

	// TOP5 project managers
	stats.Top5Managers = s.getTop5(db, "project_manager")

	// TOP5 customers (end_user)
	stats.Top5Customers = s.getTop5(db, "end_user")

	// Year-over-year comparison
	stats.YoYComparison = s.getYoYComparison(db)

	// Weekly daily breakdown
	stats.WeeklyProjects = s.getWeeklyDaily(db)

	return stats, nil
}

// getMonthlyTrend returns monthly project creation counts for the last 12 months
func (s *ProjectService) getMonthlyTrend(db *gorm.DB) []MonthlyCount {
	var results []MonthlyCount
	now := time.Now()
	startMonth := now.AddDate(-1, 0, 0).Format("2006-01")

	rows, err := db.Model(&model.ProjectInfo{}).
		Select("DATE_FORMAT(project_start_date, '%Y-%m') as month, COUNT(*) as count").
		Where("project_start_date >= ? AND project_start_date != ''", startMonth+"-01").
		Group("month").
		Order("month ASC").
		Rows()
	if err != nil {
		logger.Log.Warnf("getMonthlyTrend error: %v", err)
		return results
	}
	defer rows.Close()

	for rows.Next() {
		var mc MonthlyCount
		rows.Scan(&mc.Month, &mc.Count)
		results = append(results, mc)
	}
	return results
}

// getDistribution returns name-count distribution for a given column
func (s *ProjectService) getDistribution(db *gorm.DB, column string) []NameCount {
	var results []NameCount
	rows, err := db.Model(&model.ProjectInfo{}).
		Select(fmt.Sprintf("%s as name, COUNT(*) as count", column)).
		Where(fmt.Sprintf("%s != '' AND %s IS NOT NULL", column, column)).
		Group("name").
		Order("count DESC").
		Limit(10).
		Rows()
	if err != nil {
		logger.Log.Warnf("getDistribution(%s) error: %v", column, err)
		return results
	}
	defer rows.Close()

	for rows.Next() {
		var nc NameCount
		rows.Scan(&nc.Name, &nc.Count)
		results = append(results, nc)
	}
	return results
}

// getTop5 returns top 5 entries for a given column
func (s *ProjectService) getTop5(db *gorm.DB, column string) []NameCount {
	var results []NameCount
	rows, err := db.Model(&model.ProjectInfo{}).
		Select(fmt.Sprintf("%s as name, COUNT(*) as count", column)).
		Where(fmt.Sprintf("%s != '' AND %s IS NOT NULL", column, column)).
		Group("name").
		Order("count DESC").
		Limit(5).
		Rows()
	if err != nil {
		logger.Log.Warnf("getTop5(%s) error: %v", column, err)
		return results
	}
	defer rows.Close()

	for rows.Next() {
		var nc NameCount
		rows.Scan(&nc.Name, &nc.Count)
		results = append(results, nc)
	}
	return results
}

// getYoYComparison returns year-over-year project creation counts
func (s *ProjectService) getYoYComparison(db *gorm.DB) []YoYData {
	var results []YoYData
	rows, err := db.Model(&model.ProjectInfo{}).
		Select("YEAR(project_start_date) as year, COUNT(*) as count").
		Where("project_start_date != '' AND project_start_date IS NOT NULL AND YEAR(project_start_date) >= 2020").
		Group("year").
		Order("year ASC").
		Rows()
	if err != nil {
		logger.Log.Warnf("getYoYComparison error: %v", err)
		return results
	}
	defer rows.Close()

	for rows.Next() {
		var yd YoYData
		rows.Scan(&yd.Year, &yd.Count)
		results = append(results, yd)
	}
	return results
}

// getWeeklyDaily returns daily project creation for the past 7 days
func (s *ProjectService) getWeeklyDaily(db *gorm.DB) []DailyCount {
	var results []DailyCount
	weekAgo := time.Now().AddDate(0, 0, -7).Format("2006-01-02")

	rows, err := db.Model(&model.ProjectInfo{}).
		Select("DATE(project_start_date) as date, COUNT(*) as count").
		Where("project_start_date >= ?", weekAgo).
		Group("date").
		Order("date ASC").
		Rows()
	if err != nil {
		logger.Log.Warnf("getWeeklyDaily error: %v", err)
		return results
	}
	defer rows.Close()

	for rows.Next() {
		var dc DailyCount
		rows.Scan(&dc.Date, &dc.Count)
		results = append(results, dc)
	}
	return results
}

// GetProjectList returns paginated project list with optional filters
func (s *ProjectService) GetProjectList(page, pageSize int, search, region, projectType, status string) ([]model.ProjectInfo, int64, error) {
	var projects []model.ProjectInfo
	var total int64

	query := repository.DB.Model(&model.ProjectInfo{})

	if search != "" {
		query = query.Where("project_name LIKE ? OR project_no LIKE ? OR project_manager LIKE ? OR end_user LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if region != "" {
		query = query.Where("region = ?", region)
	}
	if projectType != "" {
		query = query.Where("project_type = ?", projectType)
	}
	if status != "" {
		query = query.Where("project_status = ?", status)
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Order("project_start_date DESC").Offset(offset).Limit(pageSize).Find(&projects).Error
	return projects, total, err
}

// StartPeriodicSync starts a goroutine that syncs project data daily at 2:00 AM
func (s *ProjectService) StartPeriodicSync() {
	go func() {
		// Initial sync on startup (after 30s delay to let DB initialize)
		time.Sleep(30 * time.Second)

		// Only sync if local DB is empty
		var count int64
		repository.DB.Model(&model.ProjectInfo{}).Count(&count)
		if count == 0 {
			logger.Log.Info("[ProjectSync] Local DB empty, performing initial sync...")
			if err := s.SyncFromRedmine(); err != nil {
				logger.Log.Warnf("[ProjectSync] Initial sync failed: %v", err)
			}
		}

		// Schedule daily sync at 2:00 AM
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 2, 0, 0, 0, now.Location())
			duration := next.Sub(now)
			logger.Log.Infof("[ProjectSync] Next sync scheduled at: %s (in %s)", next.Format("2006-01-02 15:04:05"), duration)

			time.Sleep(duration)
			logger.Log.Info("[ProjectSync] Starting daily project sync...")
			if err := s.SyncFromRedmine(); err != nil {
				logger.Log.Warnf("[ProjectSync] Daily sync failed: %v", err)
			}
		}
	}()
}

// nullStrP extracts string from sql.NullString (project-specific to avoid redeclaration)
func nullStrP(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}
