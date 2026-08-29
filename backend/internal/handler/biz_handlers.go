package handler

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jibiao-ai/deliverydesk/internal/model"
	"github.com/jibiao-ai/deliverydesk/internal/repository"
	"github.com/jibiao-ai/deliverydesk/pkg/logger"
	"github.com/xuri/excelize/v2"
)

type BizHandler struct{}

func NewBizHandler() *BizHandler {
	return &BizHandler{}
}

// UploadBizExcel parses the uploaded Excel, filters 维保/续保 rows, stores them
func (h *BizHandler) UploadBizExcel(c *gin.Context) {
	userID, _ := c.Get("userID")
	username, _ := c.Get("username")

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "请上传Excel文件"})
		return
	}
	defer file.Close()

	month := c.PostForm("month")
	if month == "" {
		// Try to extract month from filename, e.g. "维保商机数据08月.xlsx" → current year + 08
		month = time.Now().Format("2006-01")
	}

	f, err := excelize.OpenReader(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "无法解析Excel文件: " + err.Error()})
		return
	}
	defer f.Close()

	// Find the data sheet (first non-hidden sheet, or "商机数据")
	sheetName := "商机数据"
	sheets := f.GetSheetList()
	found := false
	for _, s := range sheets {
		if s == sheetName {
			found = true
			break
		}
	}
	if !found && len(sheets) > 0 {
		sheetName = sheets[0]
	}

	rows, err := f.GetRows(sheetName)
	if err != nil || len(rows) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "无法读取工作表数据"})
		return
	}

	// Parse header to find column indices
	headerRow := rows[0]
	colIdx := map[string]int{}
	colNames := []string{"商机名称", "客户名称", "商机编号", "商机总金额", "易捷行云培训及服务", "预计成交日期", "状态", "负责人", "售前人员", "负责人所属核心管控单元", "省份", "城市", "商机赢率", "总节点数", "购买类型", "创建人", "创建时间"}
	for i, cell := range headerRow {
		for _, cn := range colNames {
			if strings.Contains(cell, cn) {
				colIdx[cn] = i
				break
			}
		}
	}

	// Create upload history record
	uid, _ := userID.(uint)
	uname, _ := username.(string)
	upload := model.BizUploadHistory{
		FileName:     header.Filename,
		Month:        month,
		TotalRows:    len(rows) - 1,
		UploadedBy:   uid,
		UploadedName: uname,
	}

	// Delete old data for the same month (replace mode)
	repository.DB.Where("month = ?", month).Delete(&model.BizOpportunity{})
	repository.DB.Where("month = ?", month).Delete(&model.BizUploadHistory{})

	if err := repository.DB.Create(&upload).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "保存上传记录失败"})
		return
	}

	getCell := func(row []string, colName string) string {
		idx, ok := colIdx[colName]
		if !ok || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	var opportunities []model.BizOpportunity
	for _, row := range rows[1:] {
		name := getCell(row, "商机名称")
		if name == "" {
			continue
		}

		// Filter: only rows containing 维保 or 续保
		bizType := ""
		if strings.Contains(name, "维保") {
			bizType = "维保"
		}
		if strings.Contains(name, "续保") {
			if bizType != "" {
				bizType = "维保续保"
			} else {
				bizType = "续保"
			}
		}
		if bizType == "" {
			continue
		}

		amountStr := getCell(row, "商机总金额")
		amount, _ := strconv.ParseFloat(strings.ReplaceAll(amountStr, ",", ""), 64)
		svcAmountStr := getCell(row, "易捷行云培训及服务")
		svcAmount, _ := strconv.ParseFloat(strings.ReplaceAll(svcAmountStr, ",", ""), 64)
		nodeStr := getCell(row, "总节点数")
		nodeCount, _ := strconv.ParseFloat(nodeStr, 64)

		opp := model.BizOpportunity{
			UploadID:       upload.ID,
			Month:          month,
			Name:           name,
			Customer:       getCell(row, "客户名称"),
			Code:           getCell(row, "商机编号"),
			Amount:         amount,
			ServiceAmount:  svcAmount,
			ExpectedDate:   getCell(row, "预计成交日期"),
			Status:         getCell(row, "状态"),
			Owner:          getCell(row, "负责人"),
			PreSales:       getCell(row, "售前人员"),
			Region:         getCell(row, "负责人所属核心管控单元"),
			Province:       getCell(row, "省份"),
			City:           getCell(row, "城市"),
			WinRate:        getCell(row, "商机赢率"),
			NodeCount:      nodeCount,
			BuyType:        getCell(row, "购买类型"),
			Creator:        getCell(row, "创建人"),
			OrigCreateTime: getCell(row, "创建时间"),
			BizType:        bizType,
		}
		opportunities = append(opportunities, opp)
	}

	if len(opportunities) > 0 {
		repository.DB.CreateInBatches(&opportunities, 100)
	}

	upload.FilteredRows = len(opportunities)
	repository.DB.Save(&upload)

	logger.Log.Infof("BizUpload: %s uploaded %s, total=%d, filtered=%d", uname, header.Filename, len(rows)-1, len(opportunities))

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"upload_id":     upload.ID,
			"file_name":     header.Filename,
			"month":         month,
			"total_rows":    len(rows) - 1,
			"filtered_rows": len(opportunities),
		},
	})
}

// ListBizOpportunities returns filtered/paginated biz opportunity records
func (h *BizHandler) ListBizOpportunities(c *gin.Context) {
	month := c.Query("month")
	status := c.Query("status")
	region := c.Query("region")
	bizType := c.Query("biz_type")
	search := c.Query("search")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	query := repository.DB.Model(&model.BizOpportunity{})
	if month != "" {
		query = query.Where("month = ?", month)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if region != "" {
		query = query.Where("region = ?", region)
	}
	if bizType != "" {
		query = query.Where("biz_type LIKE ?", "%"+bizType+"%")
	}
	if search != "" {
		query = query.Where("name LIKE ? OR customer LIKE ? OR code LIKE ? OR owner LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	query.Count(&total)

	var records []model.BizOpportunity
	query.Order("amount DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&records)

	c.JSON(http.StatusOK, gin.H{
		"code":  0,
		"data":  records,
		"total": total,
		"page":  page,
	})
}

// GetBizStats returns comprehensive statistics for charts
func (h *BizHandler) GetBizStats(c *gin.Context) {
	month := c.Query("month") // optional, empty = all months

	query := repository.DB.Model(&model.BizOpportunity{})
	if month != "" {
		query = query.Where("month = ?", month)
	}

	var records []model.BizOpportunity
	query.Find(&records)

	if len(records) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"total": 0}})
		return
	}

	// Overview
	totalAmount := 0.0
	totalServiceAmount := 0.0
	totalNodes := 0.0
	statusMap := map[string]int{}
	regionAmountMap := map[string]float64{}
	regionCountMap := map[string]int{}
	ownerAmountMap := map[string]float64{}
	ownerCountMap := map[string]int{}
	// provinceAmountMap removed — TOP10 区域 now uses regionAmountMap (负责人所属核心管控单元)
	buyTypeMap := map[string]int{}
	bizTypeMap := map[string]int{}
	winRateDistMap := map[string]int{}
	monthAmountMap := map[string]float64{}
	monthCountMap := map[string]int{}

	for _, r := range records {
		totalAmount += r.Amount
		totalServiceAmount += r.ServiceAmount
		totalNodes += r.NodeCount
		statusMap[r.Status]++
		regionAmountMap[r.Region] += r.Amount
		regionCountMap[r.Region]++
		ownerAmountMap[r.Owner] += r.Amount
		ownerCountMap[r.Owner]++
		// Province aggregation removed — region_data already covers 负责人所属核心管控单元
		if r.BuyType != "" {
			buyTypeMap[r.BuyType]++
		}
		bizTypeMap[r.BizType]++
		if r.WinRate != "" {
			winRateDistMap[r.WinRate]++
		}
		monthAmountMap[r.Month] += r.Amount
		monthCountMap[r.Month]++
	}

	// TOP10 by amount
	type kv struct {
		Key   string  `json:"name"`
		Value float64 `json:"value"`
		Count int     `json:"count,omitempty"`
	}

	sortedOwners := []kv{}
	for k, v := range ownerAmountMap {
		sortedOwners = append(sortedOwners, kv{Key: k, Value: math.Round(v*100) / 100, Count: ownerCountMap[k]})
	}
	sort.Slice(sortedOwners, func(i, j int) bool { return sortedOwners[i].Value > sortedOwners[j].Value })
	top10Owners := sortedOwners
	if len(top10Owners) > 10 {
		top10Owners = top10Owners[:10]
	}

	// TOP10 customers by amount
	customerAmountMap := map[string]float64{}
	customerCountMap := map[string]int{}
	for _, r := range records {
		customerAmountMap[r.Customer] += r.Amount
		customerCountMap[r.Customer]++
	}
	sortedCustomers := []kv{}
	for k, v := range customerAmountMap {
		sortedCustomers = append(sortedCustomers, kv{Key: k, Value: math.Round(v*100) / 100, Count: customerCountMap[k]})
	}
	sort.Slice(sortedCustomers, func(i, j int) bool { return sortedCustomers[i].Value > sortedCustomers[j].Value })
	top10Customers := sortedCustomers
	if len(top10Customers) > 10 {
		top10Customers = top10Customers[:10]
	}

	// Region data for charts
	regionData := []kv{}
	for k, v := range regionAmountMap {
		regionData = append(regionData, kv{Key: k, Value: math.Round(v*100) / 100, Count: regionCountMap[k]})
	}
	sort.Slice(regionData, func(i, j int) bool { return regionData[i].Value > regionData[j].Value })

	// TOP10 Region (负责人所属核心管控单元) — reuse regionData which is already sorted desc
	top10Region := regionData
	if len(top10Region) > 10 {
		top10Region = top10Region[:10]
	}

	// Status pie data
	statusData := []kv{}
	for k, v := range statusMap {
		statusData = append(statusData, kv{Key: k, Value: float64(v)})
	}

	// BuyType pie data
	buyTypeData := []kv{}
	for k, v := range buyTypeMap {
		buyTypeData = append(buyTypeData, kv{Key: k, Value: float64(v)})
	}

	// BizType pie data
	bizTypeData := []kv{}
	for k, v := range bizTypeMap {
		bizTypeData = append(bizTypeData, kv{Key: k, Value: float64(v)})
	}

	// WinRate distribution
	winRateData := []kv{}
	for k, v := range winRateDistMap {
		winRateData = append(winRateData, kv{Key: k, Value: float64(v)})
	}

	// Monthly trend (sorted)
	monthTrend := []gin.H{}
	monthKeys := []string{}
	for k := range monthAmountMap {
		monthKeys = append(monthKeys, k)
	}
	sort.Strings(monthKeys)
	for _, mk := range monthKeys {
		monthTrend = append(monthTrend, gin.H{
			"month":  mk,
			"amount": math.Round(monthAmountMap[mk]*100) / 100,
			"count":  monthCountMap[mk],
		})
	}

	// Radar data by region (dimensions: 金额, 数量, 节点数, 服务金额, 赢率中位数)
	regionNodeMap := map[string]float64{}
	regionSvcMap := map[string]float64{}
	for _, r := range records {
		regionNodeMap[r.Region] += r.NodeCount
		regionSvcMap[r.Region] += r.ServiceAmount
	}
	radarData := []gin.H{}
	for _, rd := range regionData {
		radarData = append(radarData, gin.H{
			"region":         rd.Key,
			"amount":         rd.Value,
			"count":          rd.Count,
			"nodes":          regionNodeMap[rd.Key],
			"service_amount": math.Round(regionSvcMap[rd.Key]*100) / 100,
		})
	}

	avgAmount := 0.0
	if len(records) > 0 {
		avgAmount = math.Round(totalAmount/float64(len(records))*100) / 100
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"overview": gin.H{
				"total_count":          len(records),
				"total_amount":         math.Round(totalAmount*100) / 100,
				"total_service_amount": math.Round(totalServiceAmount*100) / 100,
				"total_nodes":          totalNodes,
				"avg_amount":           avgAmount,
				"status_in_progress":   statusMap["进行中"],
				"status_won":           statusMap["赢单"],
			},
			"top10_owners":    top10Owners,
			"top10_customers": top10Customers,
			"top10_region":    top10Region,
			"region_data":     regionData,
			"status_data":     statusData,
			"buy_type_data":   buyTypeData,
			"biz_type_data":   bizTypeData,
			"win_rate_data":   winRateData,
			"month_trend":     monthTrend,
			"radar_data":      radarData,
		},
	})
}

// ListUploadHistory returns all upload history records
func (h *BizHandler) ListUploadHistory(c *gin.Context) {
	var records []model.BizUploadHistory
	repository.DB.Order("created_at DESC").Find(&records)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": records})
}

// DeleteUpload deletes an upload and its associated opportunity records
func (h *BizHandler) DeleteUpload(c *gin.Context) {
	id := c.Param("id")
	var upload model.BizUploadHistory
	if err := repository.DB.First(&upload, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "上传记录不存在"})
		return
	}
	repository.DB.Where("upload_id = ?", upload.ID).Delete(&model.BizOpportunity{})
	repository.DB.Delete(&upload)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// GetAvailableMonths returns the list of months with uploaded data
func (h *BizHandler) GetAvailableMonths(c *gin.Context) {
	var months []string
	repository.DB.Model(&model.BizUploadHistory{}).
		Distinct("month").
		Order("month DESC").
		Pluck("month", &months)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": months})
}

// GetFilterOptions returns available filter values (status, region, etc.)
func (h *BizHandler) GetFilterOptions(c *gin.Context) {
	month := c.Query("month")
	query := repository.DB.Model(&model.BizOpportunity{})
	if month != "" {
		query = query.Where("month = ?", month)
	}

	var statuses, regions, bizTypes, buyTypes []string
	query.Distinct("status").Where("status != ''").Pluck("status", &statuses)

	query2 := repository.DB.Model(&model.BizOpportunity{})
	if month != "" {
		query2 = query2.Where("month = ?", month)
	}
	query2.Distinct("region").Where("region != ''").Pluck("region", &regions)

	query3 := repository.DB.Model(&model.BizOpportunity{})
	if month != "" {
		query3 = query3.Where("month = ?", month)
	}
	query3.Distinct("biz_type").Where("biz_type != ''").Pluck("biz_type", &bizTypes)

	query4 := repository.DB.Model(&model.BizOpportunity{})
	if month != "" {
		query4 = query4.Where("month = ?", month)
	}
	query4.Distinct("buy_type").Where("buy_type != ''").Pluck("buy_type", &buyTypes)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"statuses":  statuses,
			"regions":   regions,
			"biz_types": bizTypes,
			"buy_types": buyTypes,
		},
	})
}

// ExportBizExcel exports filtered data as Excel
func (h *BizHandler) ExportBizExcel(c *gin.Context) {
	month := c.Query("month")
	query := repository.DB.Model(&model.BizOpportunity{})
	if month != "" {
		query = query.Where("month = ?", month)
	}

	var records []model.BizOpportunity
	query.Order("amount DESC").Find(&records)

	f := excelize.NewFile()
	sheet := "维保续保商机"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"商机名称", "客户名称", "商机编号", "商机总金额", "服务金额", "预计成交日期", "状态", "负责人", "售前人员", "区域", "省份", "城市", "赢率", "节点数", "购买类型", "类型", "月份"}
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"6C5CE7"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	for i, r := range records {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), r.Name)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), r.Customer)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), r.Code)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), r.Amount)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), r.ServiceAmount)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), r.ExpectedDate)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), r.Status)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), r.Owner)
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), r.PreSales)
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), r.Region)
		f.SetCellValue(sheet, fmt.Sprintf("K%d", row), r.Province)
		f.SetCellValue(sheet, fmt.Sprintf("L%d", row), r.City)
		f.SetCellValue(sheet, fmt.Sprintf("M%d", row), r.WinRate)
		f.SetCellValue(sheet, fmt.Sprintf("N%d", row), r.NodeCount)
		f.SetCellValue(sheet, fmt.Sprintf("O%d", row), r.BuyType)
		f.SetCellValue(sheet, fmt.Sprintf("P%d", row), r.BizType)
		f.SetCellValue(sheet, fmt.Sprintf("Q%d", row), r.Month)
	}

	// Auto-set column widths
	widths := []float64{30, 25, 18, 14, 14, 14, 10, 10, 10, 14, 10, 10, 8, 8, 8, 10, 10}
	for i, w := range widths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, w)
	}

	fileName := "维保续保商机数据"
	if month != "" {
		fileName += "_" + month
	}
	fileName += ".xlsx"

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", fileName))
	f.Write(c.Writer)
}
