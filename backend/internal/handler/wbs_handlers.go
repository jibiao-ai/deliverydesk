package handler

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jibiao-ai/deliverydesk/internal/model"
	"github.com/jibiao-ai/deliverydesk/internal/repository"
	"github.com/xuri/excelize/v2"
)

//go:embed wbs_catalog_data.json
var catalogJSON embed.FS

// WBSHandler handles WBS service endpoints
type WBSHandler struct{}

// =================== Catalog Data Structures ===================

// CatalogItem represents one item in the embedded catalog JSON
type CatalogItem struct {
	Name          string `json:"name"`
	Code          string `json:"code"`
	MajorCategory string `json:"major_category"` // 自有产品 / 云平台增值软件及服务
	SubCategory   string `json:"sub_category"`
	Series        string `json:"series"`
	Description   string `json:"description"`
	Module        string `json:"module"`
	Arch          string `json:"arch"`
	BuyProduct    string `json:"buy_product"`
	LicenseType   string `json:"license_type"`
	Version       string `json:"version"`  // V611 / V621
	ItemType      string `json:"item_type"` // product / service
	Unit          string `json:"unit"`
}

// loadCatalog reads the embedded JSON catalog
func loadCatalog() ([]CatalogItem, error) {
	data, err := catalogJSON.ReadFile("wbs_catalog_data.json")
	if err != nil {
		return nil, err
	}
	var items []CatalogItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// catalogItemID generates a unique ID for a catalog item
func catalogItemID(item CatalogItem) string {
	return fmt.Sprintf("%s_%s_%s", item.Version, item.Code, item.Arch)
}

// =================== Request/Response Types ===================

// WBSCatalogResponse is the API response for catalog
type WBSCatalogResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Code          string `json:"code"`
	MajorCategory string `json:"major_category"`
	SubCategory   string `json:"sub_category"`
	Series        string `json:"series"`
	Description   string `json:"description"`
	Module        string `json:"module"`
	Arch          string `json:"arch"`
	BuyProduct    string `json:"buy_product"`
	LicenseType   string `json:"license_type"`
	Version       string `json:"version"`
	ItemType      string `json:"item_type"`
	Unit          string `json:"unit"`
}

// WBSEnvironmentReq represents environment in request
type WBSEnvironmentReq struct {
	EnvName        string `json:"env_name"`
	EnvType        string `json:"env_type"`
	ProductVersion string `json:"product_version"`
	LicenseType    string `json:"license_type"`
	ArchType       string `json:"arch_type"`
	SLA            string `json:"sla"`
	MaintenanceYr  int    `json:"maintenance_yr"`
	ChangeLogo     bool   `json:"change_logo"`
}

// WBSOrderItemReq represents an order item in request
type WBSOrderItemReq struct {
	ItemID      string `json:"item_id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	Quantity    int    `json:"quantity"`
	Unit        string `json:"unit"`
	Category    string `json:"category"`
	SubCategory string `json:"sub_category"`
	Series      string `json:"series"`
	Arch        string `json:"arch"`
	Module      string `json:"module"`
	BuyProduct  string `json:"buy_product"`
	LicenseType string `json:"license_type"`
	Description string `json:"description"`
	EnvIndex    int    `json:"env_index"` // which environment this item belongs to (1-based)
}

// WBSOpportunityReq holds opportunity info
type WBSOpportunityReq struct {
	OpportunityName string `json:"opportunity_name"`
	OpportunityNo   string `json:"opportunity_no"`
	SalesOrder      string `json:"sales_order"`
	ContractNo      string `json:"contract_no"`
	CustomerName    string `json:"customer_name"`
	Agent           string `json:"agent"`
	DeployLocation  string `json:"deploy_location"`
	SalesDirector   string `json:"sales_director"`
	SalesVP         string `json:"sales_vp"`
	Sales           string `json:"sales"`
	PreSales        string `json:"pre_sales"`
	DeliveryLeader  string `json:"delivery_leader_email"`
	ProjectManager  string `json:"project_manager_email"`
}

// WBSSaveRequest is the full WBS order data
type WBSSaveRequest struct {
	Opportunity  WBSOpportunityReq   `json:"opportunity"`
	Environments []WBSEnvironmentReq `json:"environments"`
	Products     []WBSOrderItemReq   `json:"products"`
	Services     []WBSOrderItemReq   `json:"services"`
	Remarks      string              `json:"remarks"`
}

// =================== API Handlers ===================

// GetCatalog returns the product & service catalog, optionally filtered by version
func (h *WBSHandler) GetCatalog(c *gin.Context) {
	version := c.Query("version") // V611 or V621
	items, err := loadCatalog()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "加载产品目录失败: " + err.Error()})
		return
	}

	var products []WBSCatalogResponse
	var services []WBSCatalogResponse

	for _, item := range items {
		// Filter by version if specified
		if version != "" && item.Version != version {
			continue
		}

		resp := WBSCatalogResponse{
			ID:            catalogItemID(item),
			Name:          item.Name,
			Code:          item.Code,
			MajorCategory: item.MajorCategory,
			SubCategory:   item.SubCategory,
			Series:        item.Series,
			Description:   item.Description,
			Module:        item.Module,
			Arch:          item.Arch,
			BuyProduct:    item.BuyProduct,
			LicenseType:   item.LicenseType,
			Version:       item.Version,
			ItemType:      item.ItemType,
			Unit:          item.Unit,
		}

		if item.ItemType == "product" {
			products = append(products, resp)
		} else {
			services = append(services, resp)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"products": products,
			"services": services,
		},
	})
}

// SaveOrder saves a WBS order with environments and items
func (h *WBSHandler) SaveOrder(c *gin.Context) {
	var req WBSSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误: " + err.Error()})
		return
	}

	username, _ := c.Get("username")
	userID, _ := c.Get("user_id")

	// Create order
	order := model.WBSOrder{
		UserID:          fmt.Sprintf("%v", userID),
		Username:        fmt.Sprintf("%v", username),
		OpportunityName: req.Opportunity.OpportunityName,
		OpportunityNo:   req.Opportunity.OpportunityNo,
		SalesOrder:      req.Opportunity.SalesOrder,
		ContractNo:      req.Opportunity.ContractNo,
		CustomerName:    req.Opportunity.CustomerName,
		Agent:           req.Opportunity.Agent,
		DeployLocation:  req.Opportunity.DeployLocation,
		SalesDirector:   req.Opportunity.SalesDirector,
		SalesVP:         req.Opportunity.SalesVP,
		Sales:           req.Opportunity.Sales,
		PreSales:        req.Opportunity.PreSales,
		DeliveryLeader:  req.Opportunity.DeliveryLeader,
		ProjectManager:  req.Opportunity.ProjectManager,
		ProductCount:    len(req.Products),
		ServiceCount:    len(req.Services),
		Status:          "draft",
		Remarks:         req.Remarks,
	}

	if err := repository.DB.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "保存失败: " + err.Error()})
		return
	}

	// Save environments
	var envModels []model.WBSEnvironment
	for i, env := range req.Environments {
		envModels = append(envModels, model.WBSEnvironment{
			OrderID:        order.ID,
			EnvIndex:       i + 1,
			EnvName:        env.EnvName,
			EnvType:        env.EnvType,
			ProductVersion: env.ProductVersion,
			LicenseType:    env.LicenseType,
			ArchType:       env.ArchType,
			SLA:            env.SLA,
			MaintenanceYr:  env.MaintenanceYr,
			ChangeLogo:     env.ChangeLogo,
		})
	}
	if len(envModels) > 0 {
		if err := repository.DB.Create(&envModels).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "保存环境信息失败: " + err.Error()})
			return
		}
	}

	// Build env index -> ID map
	envIDMap := make(map[int]uint)
	for _, env := range envModels {
		envIDMap[env.EnvIndex] = env.ID
	}

	// Save order items (products)
	var orderItems []model.WBSOrderItem
	for _, p := range req.Products {
		if p.Quantity > 0 {
			envID := uint(0)
			if p.EnvIndex > 0 {
				envID = envIDMap[p.EnvIndex]
			}
			orderItems = append(orderItems, model.WBSOrderItem{
				OrderID:     order.ID,
				EnvID:       envID,
				ItemType:    "product",
				ItemID:      p.ItemID,
				Name:        p.Name,
				Code:        p.Code,
				Quantity:    p.Quantity,
				Unit:        p.Unit,
				Category:    p.Category,
				SubCategory: p.SubCategory,
				Series:      p.Series,
				Arch:        p.Arch,
				Module:      p.Module,
				BuyProduct:  p.BuyProduct,
				LicenseType: p.LicenseType,
				Description: p.Description,
			})
		}
	}
	// Save order items (services)
	for _, s := range req.Services {
		if s.Quantity > 0 {
			envID := uint(0)
			if s.EnvIndex > 0 {
				envID = envIDMap[s.EnvIndex]
			}
			orderItems = append(orderItems, model.WBSOrderItem{
				OrderID:     order.ID,
				EnvID:       envID,
				ItemType:    "service",
				ItemID:      s.ItemID,
				Name:        s.Name,
				Code:        s.Code,
				Quantity:    s.Quantity,
				Unit:        s.Unit,
				Category:    s.Category,
				SubCategory: s.SubCategory,
				Series:      s.Series,
				Description: s.Description,
			})
		}
	}

	if len(orderItems) > 0 {
		if err := repository.DB.Create(&orderItems).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "保存订单项失败: " + err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"order_id":     order.ID,
			"environments": envModels,
			"items":        orderItems,
			"message":      "WBS订单保存成功",
		},
	})
}

// ListOrders returns paginated WBS orders
func (h *WBSHandler) ListOrders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}

	var total int64
	repository.DB.Model(&model.WBSOrder{}).Count(&total)

	var orders []model.WBSOrder
	repository.DB.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&orders)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"orders":    orders,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetOrder returns a specific order with environments and items
func (h *WBSHandler) GetOrder(c *gin.Context) {
	id := c.Param("id")
	var order model.WBSOrder
	if err := repository.DB.First(&order, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "订单不存在"})
		return
	}

	var envs []model.WBSEnvironment
	repository.DB.Where("order_id = ?", order.ID).Order("env_index").Find(&envs)

	var items []model.WBSOrderItem
	repository.DB.Where("order_id = ?", order.ID).Find(&items)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"order":        order,
			"environments": envs,
			"items":        items,
		},
	})
}

// ExportExcel generates and returns an Excel file for the WBS order
// following the standard WBS V3 template structure with 13 sheets
func (h *WBSHandler) ExportExcel(c *gin.Context) {
	id := c.Param("id")
	var order model.WBSOrder
	if err := repository.DB.First(&order, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "订单不存在"})
		return
	}

	var envs []model.WBSEnvironment
	repository.DB.Where("order_id = ?", order.ID).Order("env_index").Find(&envs)

	var items []model.WBSOrderItem
	repository.DB.Where("order_id = ?", order.ID).Find(&items)

	// Separate by type and environment
	var productItems, serviceItems []model.WBSOrderItem
	envItemsMap := make(map[uint][]model.WBSOrderItem) // envID -> items
	for _, item := range items {
		if item.ItemType == "product" {
			productItems = append(productItems, item)
		} else {
			serviceItems = append(serviceItems, item)
		}
		if item.EnvID > 0 {
			envItemsMap[item.EnvID] = append(envItemsMap[item.EnvID], item)
		}
	}

	// Separate own products vs value-added products
	var ownProducts, vasProducts []model.WBSOrderItem
	for _, p := range productItems {
		if p.Category == "云平台增值软件及服务" || p.SubCategory == "云平台增值软件及服务" {
			vasProducts = append(vasProducts, p)
		} else {
			ownProducts = append(ownProducts, p)
		}
	}

	// Separate standard services (自有产品标准服务) and advanced services (自有产品高级服务)
	var standardServices, advancedServices []model.WBSOrderItem
	advancedCategories := map[string]bool{
		"产品高级服务": true,
		"增值运维服务": true,
		"服务人天":    true,
		"培训服务":    true,
	}
	for _, s := range serviceItems {
		if advancedCategories[s.Category] {
			advancedServices = append(advancedServices, s)
		} else {
			standardServices = append(standardServices, s)
		}
	}

	// Generate Excel
	f := excelize.NewFile()
	defer f.Close()

	// --- Common styles ---
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#DAEEF3"}, Pattern: 1},
		Border:    []excelize.Border{{Type: "left", Style: 1, Color: "999999"}, {Type: "right", Style: 1, Color: "999999"}, {Type: "top", Style: 1, Color: "999999"}, {Type: "bottom", Style: 1, Color: "999999"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	labelStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#F2F2F2"}, Pattern: 1},
		Border:    []excelize.Border{{Type: "left", Style: 1, Color: "CCCCCC"}, {Type: "right", Style: 1, Color: "CCCCCC"}, {Type: "top", Style: 1, Color: "CCCCCC"}, {Type: "bottom", Style: 1, Color: "CCCCCC"}},
		Alignment: &excelize.Alignment{Vertical: "center"},
	})
	valueStyle, _ := f.NewStyle(&excelize.Style{
		Font:   &excelize.Font{Size: 10},
		Border: []excelize.Border{{Type: "left", Style: 1, Color: "CCCCCC"}, {Type: "right", Style: 1, Color: "CCCCCC"}, {Type: "top", Style: 1, Color: "CCCCCC"}, {Type: "bottom", Style: 1, Color: "CCCCCC"}},
	})
	hintStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 9, Color: "808080", Italic: true},
	})
	dataStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Border:    []excelize.Border{{Type: "left", Style: 1, Color: "CCCCCC"}, {Type: "right", Style: 1, Color: "CCCCCC"}, {Type: "top", Style: 1, Color: "CCCCCC"}, {Type: "bottom", Style: 1, Color: "CCCCCC"}},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
	})
	envHeaderStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#4472C4"}, Pattern: 1},
		Border:    []excelize.Border{{Type: "left", Style: 1, Color: "999999"}, {Type: "right", Style: 1, Color: "999999"}, {Type: "top", Style: 1, Color: "999999"}, {Type: "bottom", Style: 1, Color: "999999"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})

	// ========== Sheet 1: 示例使用说明 ==========
	sheetGuide := "示例使用说明"
	f.SetSheetName("Sheet1", sheetGuide)
	f.SetColWidth(sheetGuide, "A", "A", 25)
	f.SetColWidth(sheetGuide, "B", "B", 80)
	f.SetCellValue(sheetGuide, "A1", "WBS V3 使用说明")
	f.SetCellStyle(sheetGuide, "A1", "A1", titleStyle)
	guideRows := [][]string{
		{"步骤", "说明"},
		{"1. 填写商机信息", "在「0-商机」页面填写商机相关信息，包含客户名称、商机号、销售信息等"},
		{"2. 新建环境信息", "为每个部署环境配置版本、架构、维保年限、SLA等信息"},
		{"3. 选择产品", "根据环境版本选择对应的自有产品和云平台增值产品"},
		{"4. 选择服务", "选择安装部署、维保、增值运维、高级服务等"},
		{"5. 确认提交", "核对所有信息后提交生成WBS文件"},
	}
	for i, row := range guideRows {
		f.SetCellValue(sheetGuide, fmt.Sprintf("A%d", i+3), row[0])
		f.SetCellValue(sheetGuide, fmt.Sprintf("B%d", i+3), row[1])
		if i == 0 {
			f.SetCellStyle(sheetGuide, fmt.Sprintf("A%d", i+3), fmt.Sprintf("B%d", i+3), headerStyle)
		} else {
			f.SetCellStyle(sheetGuide, fmt.Sprintf("A%d", i+3), fmt.Sprintf("A%d", i+3), labelStyle)
			f.SetCellStyle(sheetGuide, fmt.Sprintf("B%d", i+3), fmt.Sprintf("B%d", i+3), dataStyle)
		}
	}

	// ========== Sheet 2: 0-商机 ==========
	sheet0 := "0-商机"
	f.NewSheet(sheet0)
	f.SetColWidth(sheet0, "A", "A", 22)
	f.SetColWidth(sheet0, "B", "B", 40)
	f.SetColWidth(sheet0, "C", "C", 40)

	f.SetCellValue(sheet0, "A1", "基础信息(售前填写）")
	f.SetCellStyle(sheet0, "A1", "A1", titleStyle)

	oppFields := []struct {
		label, value, hint string
	}{
		{"商机名称", order.OpportunityName, "提示：CRM中的商机名称"},
		{"商机号", order.OpportunityNo, "提示：CRM中的商机号（必填）"},
		{"销售订单", order.SalesOrder, "提示：如果没有就不填（如果不填项目为预交付）"},
		{"合同号", order.ContractNo, "提示：如果没有就不填（如果不填项目为预交付）"},
		{"客户名称", order.CustomerName, "提示：CRM中商机的客户名称"},
		{"代理商", order.Agent, "提示：请与CRM信息保持一致"},
		{"部署地点", order.DeployLocation, ""},
		{"销售总监", order.SalesDirector, ""},
		{"销售VP", order.SalesVP, ""},
		{"销售", order.Sales, ""},
		{"售前", order.PreSales, ""},
		{"区域交付leader邮箱", order.DeliveryLeader, "提示：要填写区域交付leader邮箱（必填）"},
		{"项目经理邮箱", order.ProjectManager, "提示：要填写项目经理邮箱（必填）"},
	}
	for i, field := range oppFields {
		row := i + 2
		f.SetCellValue(sheet0, fmt.Sprintf("A%d", row), field.label)
		f.SetCellStyle(sheet0, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), labelStyle)
		f.SetCellValue(sheet0, fmt.Sprintf("B%d", row), field.value)
		f.SetCellStyle(sheet0, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), valueStyle)
		if field.hint != "" {
			f.SetCellValue(sheet0, fmt.Sprintf("C%d", row), field.hint)
			f.SetCellStyle(sheet0, fmt.Sprintf("C%d", row), fmt.Sprintf("C%d", row), hintStyle)
		}
	}

	// ========== Sheet 3: 1-Quotation自有产品标准服务 ==========
	sheetQ1 := "1-Quotation自有产品标准服务"
	f.NewSheet(sheetQ1)
	f.SetColWidth(sheetQ1, "A", "A", 18)
	f.SetColWidth(sheetQ1, "B", "B", 45)
	f.SetColWidth(sheetQ1, "C", "C", 14)
	f.SetColWidth(sheetQ1, "D", "D", 8)
	f.SetColWidth(sheetQ1, "E", "E", 10)
	f.SetColWidth(sheetQ1, "F", "F", 60)
	q1Headers := []string{"服务类别", "服务名称", "服务编码", "数量", "单位", "服务说明"}
	for i, hdr := range q1Headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetQ1, cell, hdr)
		f.SetCellStyle(sheetQ1, cell, cell, headerStyle)
	}
	row := 2
	for _, item := range standardServices {
		f.SetCellValue(sheetQ1, fmt.Sprintf("A%d", row), item.Category)
		f.SetCellValue(sheetQ1, fmt.Sprintf("B%d", row), item.Name)
		f.SetCellValue(sheetQ1, fmt.Sprintf("C%d", row), item.Code)
		f.SetCellValue(sheetQ1, fmt.Sprintf("D%d", row), item.Quantity)
		f.SetCellValue(sheetQ1, fmt.Sprintf("E%d", row), item.Unit)
		f.SetCellValue(sheetQ1, fmt.Sprintf("F%d", row), item.Description)
		for col := 'A'; col <= 'F'; col++ {
			f.SetCellStyle(sheetQ1, fmt.Sprintf("%c%d", col, row), fmt.Sprintf("%c%d", col, row), dataStyle)
		}
		row++
	}

	// ========== Sheet 4: 2-Quotation自有产品高级服务 ==========
	sheetQ2 := "2-Quotation自有产品高级服务"
	f.NewSheet(sheetQ2)
	f.SetColWidth(sheetQ2, "A", "A", 18)
	f.SetColWidth(sheetQ2, "B", "B", 45)
	f.SetColWidth(sheetQ2, "C", "C", 14)
	f.SetColWidth(sheetQ2, "D", "D", 8)
	f.SetColWidth(sheetQ2, "E", "E", 10)
	f.SetColWidth(sheetQ2, "F", "F", 60)
	for i, hdr := range q1Headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetQ2, cell, hdr)
		f.SetCellStyle(sheetQ2, cell, cell, headerStyle)
	}
	row = 2
	for _, item := range advancedServices {
		f.SetCellValue(sheetQ2, fmt.Sprintf("A%d", row), item.Category)
		f.SetCellValue(sheetQ2, fmt.Sprintf("B%d", row), item.Name)
		f.SetCellValue(sheetQ2, fmt.Sprintf("C%d", row), item.Code)
		f.SetCellValue(sheetQ2, fmt.Sprintf("D%d", row), item.Quantity)
		f.SetCellValue(sheetQ2, fmt.Sprintf("E%d", row), item.Unit)
		f.SetCellValue(sheetQ2, fmt.Sprintf("F%d", row), item.Description)
		for col := 'A'; col <= 'F'; col++ {
			f.SetCellStyle(sheetQ2, fmt.Sprintf("%c%d", col, row), fmt.Sprintf("%c%d", col, row), dataStyle)
		}
		row++
	}

	// ========== Sheet 5: 3-Quotation云平台增值软件及服务 ==========
	sheetQ3 := "3-Quotation云平台增值软件及服务"
	f.NewSheet(sheetQ3)
	f.SetColWidth(sheetQ3, "A", "A", 20)
	f.SetColWidth(sheetQ3, "B", "B", 28)
	f.SetColWidth(sheetQ3, "C", "C", 45)
	f.SetColWidth(sheetQ3, "D", "D", 14)
	f.SetColWidth(sheetQ3, "E", "E", 8)
	f.SetColWidth(sheetQ3, "F", "F", 50)
	vasHeaders := []string{"产品大类", "产品系列", "产品名称", "产品编码", "数量", "产品说明"}
	for i, hdr := range vasHeaders {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetQ3, cell, hdr)
		f.SetCellStyle(sheetQ3, cell, cell, headerStyle)
	}
	row = 2
	for _, item := range vasProducts {
		f.SetCellValue(sheetQ3, fmt.Sprintf("A%d", row), item.SubCategory)
		f.SetCellValue(sheetQ3, fmt.Sprintf("B%d", row), item.Series)
		f.SetCellValue(sheetQ3, fmt.Sprintf("C%d", row), item.Name)
		f.SetCellValue(sheetQ3, fmt.Sprintf("D%d", row), item.Code)
		f.SetCellValue(sheetQ3, fmt.Sprintf("E%d", row), item.Quantity)
		f.SetCellValue(sheetQ3, fmt.Sprintf("F%d", row), item.Description)
		for col := 'A'; col <= 'F'; col++ {
			f.SetCellStyle(sheetQ3, fmt.Sprintf("%c%d", col, row), fmt.Sprintf("%c%d", col, row), dataStyle)
		}
		row++
	}

	// ========== Sheet 6: 4-order自有产品汇总 ==========
	sheetProd := "4-order自有产品汇总"
	f.NewSheet(sheetProd)
	prodHeaders := []string{"产品大类", "产品系列（发票开票名字）", "产品名称", "产品编码", "数量", "产品说明", "模块", "架构类型", "购买产品", "license授权类型", "产品类别"}
	for i, hdr := range prodHeaders {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetProd, cell, hdr)
		f.SetCellStyle(sheetProd, cell, cell, headerStyle)
	}
	f.SetColWidth(sheetProd, "A", "A", 20)
	f.SetColWidth(sheetProd, "B", "B", 28)
	f.SetColWidth(sheetProd, "C", "C", 45)
	f.SetColWidth(sheetProd, "D", "D", 14)
	f.SetColWidth(sheetProd, "E", "E", 8)
	f.SetColWidth(sheetProd, "F", "F", 50)
	f.SetColWidth(sheetProd, "G", "G", 12)
	f.SetColWidth(sheetProd, "H", "H", 10)
	f.SetColWidth(sheetProd, "I", "I", 14)
	f.SetColWidth(sheetProd, "J", "J", 22)
	f.SetColWidth(sheetProd, "K", "K", 10)

	row = 2
	for _, item := range ownProducts {
		f.SetCellValue(sheetProd, fmt.Sprintf("A%d", row), item.SubCategory)
		f.SetCellValue(sheetProd, fmt.Sprintf("B%d", row), item.Series)
		f.SetCellValue(sheetProd, fmt.Sprintf("C%d", row), item.Name)
		f.SetCellValue(sheetProd, fmt.Sprintf("D%d", row), item.Code)
		f.SetCellValue(sheetProd, fmt.Sprintf("E%d", row), item.Quantity)
		f.SetCellValue(sheetProd, fmt.Sprintf("F%d", row), item.Description)
		f.SetCellValue(sheetProd, fmt.Sprintf("G%d", row), item.Module)
		f.SetCellValue(sheetProd, fmt.Sprintf("H%d", row), item.Arch)
		f.SetCellValue(sheetProd, fmt.Sprintf("I%d", row), item.BuyProduct)
		f.SetCellValue(sheetProd, fmt.Sprintf("J%d", row), item.LicenseType)
		f.SetCellValue(sheetProd, fmt.Sprintf("K%d", row), item.Category)
		for col := 'A'; col <= 'K'; col++ {
			f.SetCellStyle(sheetProd, fmt.Sprintf("%c%d", col, row), fmt.Sprintf("%c%d", col, row), dataStyle)
		}
		row++
	}

	// ========== Sheet 7: 5-order自有产品按环境 ==========
	sheetProdEnv := "5-order自有产品按环境"
	f.NewSheet(sheetProdEnv)
	f.SetColWidth(sheetProdEnv, "A", "A", 20)
	f.SetColWidth(sheetProdEnv, "B", "B", 28)
	f.SetColWidth(sheetProdEnv, "C", "C", 45)
	f.SetColWidth(sheetProdEnv, "D", "D", 14)
	f.SetColWidth(sheetProdEnv, "E", "E", 8)
	f.SetColWidth(sheetProdEnv, "F", "F", 50)

	row = 1
	for _, env := range envs {
		// Environment header
		f.SetCellValue(sheetProdEnv, fmt.Sprintf("A%d", row), env.EnvName)
		f.SetCellStyle(sheetProdEnv, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), titleStyle)
		row++

		// Environment config
		envConfigHeaders := []string{"状态", "购买产品", "维保年限", "SLA", "license授权类型", "license架构类型"}
		for i, hdr := range envConfigHeaders {
			cell := fmt.Sprintf("%c%d", 'A'+i, row)
			f.SetCellValue(sheetProdEnv, cell, hdr)
			f.SetCellStyle(sheetProdEnv, cell, cell, envHeaderStyle)
		}
		row++
		f.SetCellValue(sheetProdEnv, fmt.Sprintf("A%d", row), env.EnvType)
		f.SetCellValue(sheetProdEnv, fmt.Sprintf("B%d", row), env.ProductVersion)
		f.SetCellValue(sheetProdEnv, fmt.Sprintf("C%d", row), env.MaintenanceYr)
		f.SetCellValue(sheetProdEnv, fmt.Sprintf("D%d", row), env.SLA)
		f.SetCellValue(sheetProdEnv, fmt.Sprintf("E%d", row), env.LicenseType)
		f.SetCellValue(sheetProdEnv, fmt.Sprintf("F%d", row), env.ArchType)
		for col := 'A'; col <= 'F'; col++ {
			f.SetCellStyle(sheetProdEnv, fmt.Sprintf("%c%d", col, row), fmt.Sprintf("%c%d", col, row), dataStyle)
		}
		row++
		row++ // blank line

		// Products for this environment
		prodEnvHeaders := []string{"产品大类", "产品系列", "产品名称", "产品编码", "数量", "产品说明"}
		for i, hdr := range prodEnvHeaders {
			cell := fmt.Sprintf("%c%d", 'A'+i, row)
			f.SetCellValue(sheetProdEnv, cell, hdr)
			f.SetCellStyle(sheetProdEnv, cell, cell, headerStyle)
		}
		row++

		envItems := envItemsMap[env.ID]
		for _, item := range envItems {
			if item.ItemType != "product" {
				continue
			}
			if item.Category == "云平台增值软件及服务" || item.SubCategory == "云平台增值软件及服务" {
				continue
			}
			f.SetCellValue(sheetProdEnv, fmt.Sprintf("A%d", row), item.SubCategory)
			f.SetCellValue(sheetProdEnv, fmt.Sprintf("B%d", row), item.Series)
			f.SetCellValue(sheetProdEnv, fmt.Sprintf("C%d", row), item.Name)
			f.SetCellValue(sheetProdEnv, fmt.Sprintf("D%d", row), item.Code)
			f.SetCellValue(sheetProdEnv, fmt.Sprintf("E%d", row), item.Quantity)
			f.SetCellValue(sheetProdEnv, fmt.Sprintf("F%d", row), item.Description)
			for col := 'A'; col <= 'F'; col++ {
				f.SetCellStyle(sheetProdEnv, fmt.Sprintf("%c%d", col, row), fmt.Sprintf("%c%d", col, row), dataStyle)
			}
			row++
		}
		row++ // blank between environments
	}

	// ========== Sheet 8: 6-order云平台增值软件及服务汇总 ==========
	sheetVAS := "6-order云平台增值软件及服务汇总"
	f.NewSheet(sheetVAS)
	f.SetColWidth(sheetVAS, "A", "A", 20)
	f.SetColWidth(sheetVAS, "B", "B", 28)
	f.SetColWidth(sheetVAS, "C", "C", 45)
	f.SetColWidth(sheetVAS, "D", "D", 14)
	f.SetColWidth(sheetVAS, "E", "E", 8)
	f.SetColWidth(sheetVAS, "F", "F", 50)
	for i, hdr := range vasHeaders {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetVAS, cell, hdr)
		f.SetCellStyle(sheetVAS, cell, cell, headerStyle)
	}
	row = 2
	for _, item := range vasProducts {
		f.SetCellValue(sheetVAS, fmt.Sprintf("A%d", row), item.SubCategory)
		f.SetCellValue(sheetVAS, fmt.Sprintf("B%d", row), item.Series)
		f.SetCellValue(sheetVAS, fmt.Sprintf("C%d", row), item.Name)
		f.SetCellValue(sheetVAS, fmt.Sprintf("D%d", row), item.Code)
		f.SetCellValue(sheetVAS, fmt.Sprintf("E%d", row), item.Quantity)
		f.SetCellValue(sheetVAS, fmt.Sprintf("F%d", row), item.Description)
		for col := 'A'; col <= 'F'; col++ {
			f.SetCellStyle(sheetVAS, fmt.Sprintf("%c%d", col, row), fmt.Sprintf("%c%d", col, row), dataStyle)
		}
		row++
	}

	// ========== Sheet 9: 7-order云平台增值软件及服务按环境 ==========
	sheetVASEnv := "7-order云平台增值软件及服务按环境"
	f.NewSheet(sheetVASEnv)
	f.SetColWidth(sheetVASEnv, "A", "A", 20)
	f.SetColWidth(sheetVASEnv, "B", "B", 28)
	f.SetColWidth(sheetVASEnv, "C", "C", 45)
	f.SetColWidth(sheetVASEnv, "D", "D", 14)
	f.SetColWidth(sheetVASEnv, "E", "E", 8)
	f.SetColWidth(sheetVASEnv, "F", "F", 50)

	row = 1
	for _, env := range envs {
		f.SetCellValue(sheetVASEnv, fmt.Sprintf("A%d", row), env.EnvName)
		f.SetCellStyle(sheetVASEnv, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), titleStyle)
		row++

		for i, hdr := range vasHeaders {
			cell := fmt.Sprintf("%c%d", 'A'+i, row)
			f.SetCellValue(sheetVASEnv, cell, hdr)
			f.SetCellStyle(sheetVASEnv, cell, cell, headerStyle)
		}
		row++

		envItems := envItemsMap[env.ID]
		for _, item := range envItems {
			if item.ItemType != "product" {
				continue
			}
			if item.Category != "云平台增值软件及服务" && item.SubCategory != "云平台增值软件及服务" {
				continue
			}
			f.SetCellValue(sheetVASEnv, fmt.Sprintf("A%d", row), item.SubCategory)
			f.SetCellValue(sheetVASEnv, fmt.Sprintf("B%d", row), item.Series)
			f.SetCellValue(sheetVASEnv, fmt.Sprintf("C%d", row), item.Name)
			f.SetCellValue(sheetVASEnv, fmt.Sprintf("D%d", row), item.Code)
			f.SetCellValue(sheetVASEnv, fmt.Sprintf("E%d", row), item.Quantity)
			f.SetCellValue(sheetVASEnv, fmt.Sprintf("F%d", row), item.Description)
			for col := 'A'; col <= 'F'; col++ {
				f.SetCellStyle(sheetVASEnv, fmt.Sprintf("%c%d", col, row), fmt.Sprintf("%c%d", col, row), dataStyle)
			}
			row++
		}
		row++ // blank between environments
	}

	// ========== Sheet 10: 6-order立项信息 ==========
	sheetProject := "6-order立项信息"
	f.NewSheet(sheetProject)
	f.SetCellValue(sheetProject, "A1", "项目信息")
	f.SetCellStyle(sheetProject, "A1", "A1", titleStyle)

	projHeaders := []string{"销售总监", "销售VP", "项目经理邮箱", "区域交付leader邮箱"}
	for i, hdr := range projHeaders {
		cell := fmt.Sprintf("%c2", 'A'+i)
		f.SetCellValue(sheetProject, cell, hdr)
		f.SetCellStyle(sheetProject, cell, cell, headerStyle)
	}
	f.SetCellValue(sheetProject, "A3", order.SalesDirector)
	f.SetCellValue(sheetProject, "B3", order.SalesVP)
	f.SetCellValue(sheetProject, "C3", order.ProjectManager)
	f.SetCellValue(sheetProject, "D3", order.DeliveryLeader)
	for col := 'A'; col <= 'D'; col++ {
		f.SetCellStyle(sheetProject, fmt.Sprintf("%c3", col), fmt.Sprintf("%c3", col), dataStyle)
	}
	f.SetColWidth(sheetProject, "A", "A", 14)
	f.SetColWidth(sheetProject, "B", "B", 14)
	f.SetColWidth(sheetProject, "C", "C", 25)
	f.SetColWidth(sheetProject, "D", "D", 28)

	// Environment info
	f.SetCellValue(sheetProject, "A5", "环境信息")
	f.SetCellStyle(sheetProject, "A5", "A5", titleStyle)
	envHeaders := []string{"环境名称", "状态", "购买产品", "维保年限", "SLA", "license授权类型", "license架构类型", "是否更换Logo"}
	for i, hdr := range envHeaders {
		cell := fmt.Sprintf("%c6", 'A'+i)
		f.SetCellValue(sheetProject, cell, hdr)
		f.SetCellStyle(sheetProject, cell, cell, headerStyle)
	}
	f.SetColWidth(sheetProject, "E", "E", 10)
	f.SetColWidth(sheetProject, "F", "F", 22)
	f.SetColWidth(sheetProject, "G", "G", 14)
	f.SetColWidth(sheetProject, "H", "H", 14)

	for i, env := range envs {
		r := i + 7
		f.SetCellValue(sheetProject, fmt.Sprintf("A%d", r), env.EnvName)
		f.SetCellValue(sheetProject, fmt.Sprintf("B%d", r), env.EnvType)
		f.SetCellValue(sheetProject, fmt.Sprintf("C%d", r), env.ProductVersion)
		f.SetCellValue(sheetProject, fmt.Sprintf("D%d", r), env.MaintenanceYr)
		f.SetCellValue(sheetProject, fmt.Sprintf("E%d", r), env.SLA)
		f.SetCellValue(sheetProject, fmt.Sprintf("F%d", r), env.LicenseType)
		f.SetCellValue(sheetProject, fmt.Sprintf("G%d", r), env.ArchType)
		logoStr := "否"
		if env.ChangeLogo {
			logoStr = "是"
		}
		f.SetCellValue(sheetProject, fmt.Sprintf("H%d", r), logoStr)
		for col := 'A'; col <= 'H'; col++ {
			f.SetCellStyle(sheetProject, fmt.Sprintf("%c%d", col, r), fmt.Sprintf("%c%d", col, r), dataStyle)
		}
	}

	// ========== Sheet 11: 整体模板 ==========
	sheetOverall := "整体模板"
	f.NewSheet(sheetOverall)
	f.SetCellValue(sheetOverall, "A1", "通用")
	f.SetCellValue(sheetOverall, "B1", "请填写人自行核对硬件兼容性及规格信息")
	f.SetCellStyle(sheetOverall, "A1", "A1", labelStyle)
	f.SetColWidth(sheetOverall, "A", "A", 8)
	f.SetColWidth(sheetOverall, "B", "B", 50)

	// Drop-down reference data
	f.SetCellValue(sheetOverall, "U1", "购买产品")
	f.SetCellValue(sheetOverall, "U2", "ECF V612")
	f.SetCellValue(sheetOverall, "U3", "ECNF V612")
	f.SetCellValue(sheetOverall, "U4", "ECF V611")
	f.SetCellValue(sheetOverall, "U5", "ECNF V611")
	f.SetCellValue(sheetOverall, "W1", "license授权类型")
	f.SetCellValue(sheetOverall, "W2", "正式（软件永久许可）")
	f.SetCellValue(sheetOverall, "W3", "正式（软件订阅）")
	f.SetCellValue(sheetOverall, "W4", "预交付")
	f.SetCellValue(sheetOverall, "W5", "POC")
	f.SetCellValue(sheetOverall, "Y1", "license架构类型")
	f.SetCellValue(sheetOverall, "Y2", "X86")
	f.SetCellValue(sheetOverall, "Y3", "Arm")
	f.SetCellValue(sheetOverall, "AA1", "状态")
	f.SetCellValue(sheetOverall, "AA2", "新建")
	f.SetCellValue(sheetOverall, "AA3", "扩容")
	f.SetCellValue(sheetOverall, "AA4", "纯服务")
	f.SetCellValue(sheetOverall, "AA5", "升级")
	f.SetCellValue(sheetOverall, "AC1", "SLA")
	f.SetCellValue(sheetOverall, "AC2", "7x24")
	f.SetCellValue(sheetOverall, "AC3", "5x9")

	// Template structure per environment
	row = 3
	f.SetCellValue(sheetOverall, fmt.Sprintf("A%d", row), "每套环境模板结构")
	f.SetCellStyle(sheetOverall, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), titleStyle)
	row++
	tmplFields := []string{
		"环境名称: 第N套环境",
		"状态: 新建/扩容/纯服务/升级",
		"购买产品: ECF V612 / ECNF V612 / ECF V611 / ECNF V611",
		"维保年限: N年",
		"SLA: 7x24 / 5x9",
		"license授权类型: 正式（软件永久许可）/正式（软件订阅）/预交付/POC",
		"license架构类型: X86/Arm",
	}
	for _, field := range tmplFields {
		f.SetCellValue(sheetOverall, fmt.Sprintf("A%d", row), field)
		row++
	}

	// ========== Sheet 12: order模板 ==========
	sheetTmpl := "order模板"
	f.NewSheet(sheetTmpl)
	f.SetCellValue(sheetTmpl, "A1", "order模板 - 按环境汇总产品")
	f.SetCellStyle(sheetTmpl, "A1", "A1", titleStyle)
	f.SetCellValue(sheetTmpl, "A3", "环境配置区域")
	f.SetCellStyle(sheetTmpl, "A3", "A3", labelStyle)
	tmplEnvHeaders := []string{"状态", "购买产品", "维保年限", "SLA", "license授权类型", "license架构类型"}
	for i, hdr := range tmplEnvHeaders {
		cell := fmt.Sprintf("%c4", 'A'+i)
		f.SetCellValue(sheetTmpl, cell, hdr)
		f.SetCellStyle(sheetTmpl, cell, cell, headerStyle)
	}
	f.SetColWidth(sheetTmpl, "A", "A", 20)
	f.SetColWidth(sheetTmpl, "B", "B", 20)
	f.SetColWidth(sheetTmpl, "C", "C", 12)
	f.SetColWidth(sheetTmpl, "D", "D", 10)
	f.SetColWidth(sheetTmpl, "E", "E", 22)
	f.SetColWidth(sheetTmpl, "F", "F", 14)

	f.SetCellValue(sheetTmpl, "A6", "产品列表区域")
	f.SetCellStyle(sheetTmpl, "A6", "A6", labelStyle)
	tmplItemHeaders := []string{"产品大类", "产品系列", "产品名称", "产品编码", "数量", "产品说明"}
	for i, hdr := range tmplItemHeaders {
		cell := fmt.Sprintf("%c7", 'A'+i)
		f.SetCellValue(sheetTmpl, cell, hdr)
		f.SetCellStyle(sheetTmpl, cell, cell, headerStyle)
	}

	// ========== Sheet 13: 第三方对接模板 ==========
	sheet3rd := "第三方对接模板"
	f.NewSheet(sheet3rd)
	f.SetCellValue(sheet3rd, "A1", "第三方产品对接信息")
	f.SetCellStyle(sheet3rd, "A1", "A1", titleStyle)
	f.SetColWidth(sheet3rd, "A", "A", 20)
	f.SetColWidth(sheet3rd, "B", "B", 30)
	f.SetColWidth(sheet3rd, "C", "C", 45)
	f.SetColWidth(sheet3rd, "D", "D", 14)
	f.SetColWidth(sheet3rd, "E", "E", 8)
	f.SetColWidth(sheet3rd, "F", "F", 50)

	thirdPartyHeaders := []string{"产品类别", "产品系列", "产品名称", "产品编码", "数量", "产品说明"}
	for i, hdr := range thirdPartyHeaders {
		cell := fmt.Sprintf("%c2", 'A'+i)
		f.SetCellValue(sheet3rd, cell, hdr)
		f.SetCellStyle(sheet3rd, cell, cell, headerStyle)
	}
	// Third-party products are the 云平台增值软件及服务 items that have non-ECF/ECNF buy_product
	row = 3
	for _, item := range vasProducts {
		bp := item.BuyProduct
		if bp != "" && !strings.Contains(bp, "ECF") && !strings.Contains(bp, "ECNF") {
			f.SetCellValue(sheet3rd, fmt.Sprintf("A%d", row), item.SubCategory)
			f.SetCellValue(sheet3rd, fmt.Sprintf("B%d", row), item.Series)
			f.SetCellValue(sheet3rd, fmt.Sprintf("C%d", row), item.Name)
			f.SetCellValue(sheet3rd, fmt.Sprintf("D%d", row), item.Code)
			f.SetCellValue(sheet3rd, fmt.Sprintf("E%d", row), item.Quantity)
			f.SetCellValue(sheet3rd, fmt.Sprintf("F%d", row), item.Description)
			for col := 'A'; col <= 'F'; col++ {
				f.SetCellStyle(sheet3rd, fmt.Sprintf("%c%d", col, row), fmt.Sprintf("%c%d", col, row), dataStyle)
			}
			row++
		}
	}

	// Set active sheet
	idx, _ := f.GetSheetIndex(sheet0)
	f.SetActiveSheet(idx)

	// Write to response
	filename := fmt.Sprintf("WBS_%s_%s.xlsx", order.CustomerName, time.Now().Format("20060102"))
	filename = strings.ReplaceAll(filename, " ", "_")
	filename = strings.ReplaceAll(filename, "/", "_")

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Transfer-Encoding", "binary")

	if err := f.Write(c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "生成Excel失败: " + err.Error()})
	}
}

// DeleteOrder deletes a WBS order with its environments and items
func (h *WBSHandler) DeleteOrder(c *gin.Context) {
	id := c.Param("id")
	var order model.WBSOrder
	if err := repository.DB.First(&order, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "订单不存在"})
		return
	}

	// Delete items
	repository.DB.Where("order_id = ?", order.ID).Delete(&model.WBSOrderItem{})
	// Delete environments
	repository.DB.Where("order_id = ?", order.ID).Delete(&model.WBSEnvironment{})
	// Delete order
	repository.DB.Delete(&order)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}
