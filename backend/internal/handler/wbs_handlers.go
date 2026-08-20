package handler

import (
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

// WBSHandler handles WBS service endpoints
type WBSHandler struct{}

// =================== Product/Service Catalog Data ===================

// WBSProduct represents a product item in the catalog
type WBSProduct struct {
	ID          string `json:"id"`
	Category    string `json:"category"`     // 产品大类: ECF/ECNF
	Series      string `json:"series"`       // 产品系列
	Name        string `json:"name"`         // 产品名称
	Code        string `json:"code"`         // 产品编码
	Description string `json:"description"`  // 产品说明
	Module      string `json:"module"`       // 模块: 基础模块/可选模块
	Arch        string `json:"arch"`         // 架构: X86/Arm
	Product     string `json:"product"`      // 购买产品: ECF V611/ECNF V611
	LicenseType string `json:"license_type"` // license授权类型
	TypeClass   string `json:"type_class"`   // A/B1/B2
	Unit        string `json:"unit"`         // 计量单位
}

// WBSService represents a service item in the catalog
type WBSService struct {
	ID          string `json:"id"`
	Category    string `json:"category"`    // 服务类别
	Name        string `json:"name"`        // 服务名称
	Code        string `json:"code"`        // 服务编码
	Description string `json:"description"` // 服务说明
	Unit        string `json:"unit"`        // 计量单位: 套/年/人天/人次
}

// getProductCatalog returns the full product catalog
func getProductCatalog() []WBSProduct {
	return []WBSProduct{
		// ===== ECF x86 产品 =====
		{ID: "ecf-x86-eos", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "x86 数字原生引擎EOS永久许可-每CPU", Code: "11EO6000011", Description: "提供x86架构数字原生引擎EOS基础能力，包含Cloud Linux（ESCL）云操作系统、微服务编排系统EKS及软件定义计算SDC引擎、分布式存储ESS引擎、软件定义网络ENS引擎，含1年7x24电话邮件支持服务和1年EOS版本升级能力，按物理CPU颗数购买。", Module: "基础模块", Arch: "X86", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "A", Unit: "每CPU"},
		{ID: "ecf-x86-ctrl", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "x86控制平面永久许可-每CPU", Code: "11EF6009UP5", Description: "提供x86架构数字原生引擎EOS基础能力之上的计算、存储、网络、监控、管理、运维服务的统一控制平面服务。", Module: "基础模块", Arch: "X86", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "A", Unit: "每CPU"},
		{ID: "ecf-x86-compute-32", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "x86 计算服务云产品永久许可-每CPU≤32核", Code: "11EP6001011", Description: "提供x86架构软件定义的计算服务，主机高可用服务，按CPU物理颗数购买，节点CPU物理核数不超过32核。", Module: "基础模块", Arch: "X86", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "A", Unit: "每CPU"},
		{ID: "ecf-x86-compute-32p", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "x86 计算服务云产品永久许可-每CPU＞32核", Code: "11EP6001012", Description: "提供x86架构软件定义的计算服务，主机高可用服务，按CPU物理颗数购买，节点CPU物理核数超过32核。", Module: "基础模块", Arch: "X86", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "A", Unit: "每CPU"},
		{ID: "ecf-x86-block-80", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "x86 块存储云产品永久许可-每节点≤80TB", Code: "11EP6002011", Description: "提供x86架构软件定义的存储服务，管理每节点不超过80TB裸容量磁盘，按照节点数目购买。", Module: "基础模块", Arch: "X86", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "A", Unit: "每节点"},
		{ID: "ecf-x86-block-160", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "x86 块存储云产品永久许可-每节点≤160TB", Code: "11EP6002012", Description: "提供x86架构软件定义的存储服务，管理每节点不超过160TB裸容量磁盘，按照节点数目购买。", Module: "基础模块", Arch: "X86", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "A", Unit: "每节点"},
		{ID: "ecf-x86-block-160p", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "x86 块存储云产品永久许可-每节点＞160TB", Code: "11EP6002013", Description: "提供x86架构软件定义的存储服务，管理每节点超过160TB裸容量磁盘，按照节点数目购买。", Module: "基础模块", Arch: "X86", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "A", Unit: "每节点"},
		{ID: "ecf-x86-baremetal", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "x86 裸金属云产品永久许可-每节点", Code: "11EP6001015", Description: "提供x86架构裸金属主机的发放及全生命周期管理，按照节点数目购买。", Module: "基础模块", Arch: "X86", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "A", Unit: "每节点"},
		{ID: "ecf-x86-k8s", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "x86 Kubernetes容器服务云产品永久许可-每实例", Code: "11EP600A011", Description: "提供x86平台Kubernetes容器集群管理服务，按照容器宿主云主机节点数目购买。", Module: "基础模块", Arch: "X86", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "B1", Unit: "每实例"},
		{ID: "ecf-x86-hps-compute", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "x86 高性能块存储计算侧加速云产品永久许可-25TB", Code: "11EP6002031", Description: "存储产品线高性能存储v6.0.2版本，提供x86架构软件定义的计算侧加速的高性能低延迟存储服务。", Module: "基础模块", Arch: "X86", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "B1", Unit: "每25TB"},
		{ID: "ecf-x86-hps-storage", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "x86 高性能块存储存储侧加速云产品永久许可-30TB", Code: "11EP6002032", Description: "存储产品线高性能存储v6.1.1版本，提供x86架构软件定义的存储侧加速的高性能低延迟存储服务，最小起配30TB。", Module: "基础模块", Arch: "X86", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "B1", Unit: "每30TB"},
		{ID: "ecf-x86-sdn-basic", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "x86 SDN基础网络服务云产品永久许可", Code: "11EP6004020", Description: "SDN基础网络服务云产品只支持采用二层组网的传统网络模型，按套收费。", Module: "基础模块", Arch: "X86", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "B1", Unit: "每套"},
		{ID: "ecf-x86-sdn-adv", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "x86 SDN高级网络服务云产品永久许可", Code: "11EP6004013", Description: "SDN高级网络服务云产品支持传统网络、路由网络及标准网络三种网络模型，按套收费。", Module: "基础模块", Arch: "X86", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "B1", Unit: "每套"},
		{ID: "ecf-x86-sdn-ext", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "x86 SDN基础网络服务云产品扩展包永久许可", Code: "11EP6004021", Description: "在SDN基础网络服务云产品基础上增加三层路由器功能，按套收费。", Module: "基础模块", Arch: "X86", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "B1", Unit: "每套"},
		{ID: "ecf-x86-acl-fw", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "x86 网络ACL和网关防火墙云产品永久许可", Code: "11EP6004022", Description: "提供网络ACL和网关防火墙功能，按套收费。", Module: "基础模块", Arch: "X86", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "B1", Unit: "每套"},
		{ID: "ecf-x86-net-node", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "x86 计算节点网络服务云产品永久许可-每节点", Code: "11EP6004016", Description: "提供x86架构软件定义的计算节点网络服务云产品，按照节点数目购买。", Module: "基础模块", Arch: "X86", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "随SDN", Unit: "每节点"},
		{ID: "ecf-x86-devops-unlim", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "x86 Devops云产品永久许可-不限流水线", Code: "11EP600A013", Description: "DevOps自动交付流水线，不限流水线数量。", Module: "基础模块", Arch: "X86", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "B1", Unit: "每套"},
		{ID: "ecf-x86-lb", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "x86 独享型负载均衡云产品永久许可-每32vCPU逻辑核", Code: "11EP6004015", Description: "提供x86架构高可靠的四层、七层负载均衡服务，按部署实例vCPU数目购买。", Module: "基础模块", Arch: "X86", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "B1", Unit: "每32vCPU"},
		{ID: "ecf-x86-multizone", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "x86 多区域管理云产品永久许可-每区域", Code: "11EP600F011", Description: "提供x86平台管理多个计算/存储资源区域服务，按照区域总数购买。", Module: "基础模块", Arch: "X86", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "B1", Unit: "每区域"},
		{ID: "ecf-x86-multiarch", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "x86 一云多芯云产品永久许可", Code: "11EP6001014", Description: "提供在x86平台同一个区域内支持Arm计算节点的能力。", Module: "基础模块", Arch: "X86", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "B1", Unit: "每套"},
		{ID: "ecf-x86-gpu", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "x86 GPU调度云产品永久许可-每16*GPU", Code: "11EP6001013", Description: "提供GPU智能调度服务，按照16块GPU一包购买。", Module: "基础模块", Arch: "X86", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "B1", Unit: "每16GPU"},
		{ID: "ecf-x86-cert", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "x86 证书和密钥云产品永久许可", Code: "11EP600B012", Description: "提供x86架构下证书与密钥服务。", Module: "基础模块", Arch: "X86", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "B1", Unit: "每套"},
		{ID: "ecf-x86-3rd-storage", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "x86 对接商业存储模块永久许可", Code: "11EP6002014", Description: "提供x86平台对接商业存储服务，按照商业存储套数购买。", Module: "基础模块", Arch: "X86", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "B2", Unit: "每套"},
		{ID: "ecf-x86-ldap", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "x86 LDAP对接永久许可", Code: "11EP600B011", Description: "提供x86 LDAP对接，按照功能开通购买。", Module: "基础模块", Arch: "X86", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "B2", Unit: "每套"},
		{ID: "ecf-x86-sdn-hw", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "x86 对接SDN模块永久许可", Code: "11EP6004011", Description: "提供x86平台对接SDN控制器服务，按照功能开通购买。", Module: "基础模块", Arch: "X86", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "B2", Unit: "每套"},
		// ===== ECF Arm 产品 =====
		{ID: "ecf-arm-eos", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "Arm 数字原生引擎EOS永久许可-每CPU", Code: "11EO6020011", Description: "提供Arm架构数字原生引擎EOS基础能力，按物理CPU颗数购买。", Module: "基础模块", Arch: "Arm", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "A", Unit: "每CPU"},
		{ID: "ecf-arm-ctrl", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "Arm 控制平面永久许可-每CPU", Code: "11EF6009UP5", Description: "提供Arm架构统一控制平面服务，按物理CPU颗数购买。最小起配为3节点。", Module: "基础模块", Arch: "Arm", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "A", Unit: "每CPU"},
		{ID: "ecf-arm-compute-48", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "Arm 计算服务云产品永久许可-每CPU≤48核", Code: "11EP6021011", Description: "提供Arm架构软件定义的计算服务，按CPU物理颗数购买，节点CPU物理核数不超过48核。", Module: "基础模块", Arch: "Arm", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "A", Unit: "每CPU"},
		{ID: "ecf-arm-compute-48p", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "Arm 计算服务云产品永久许可-每CPU＞48核", Code: "11EP6021012", Description: "提供Arm架构软件定义的计算服务，按CPU物理颗数购买，节点CPU物理核数超过48核。", Module: "基础模块", Arch: "Arm", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "A", Unit: "每CPU"},
		{ID: "ecf-arm-block-80", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "Arm 块存储云产品永久许可-每节点≤80TB", Code: "11EP6022011", Description: "提供Arm架构软件定义的存储服务，管理每节点不超过80TB裸容量磁盘，按照节点数目购买。", Module: "基础模块", Arch: "Arm", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "A", Unit: "每节点"},
		{ID: "ecf-arm-block-160", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "Arm 块存储云产品永久许可-每节点≤160TB", Code: "11EP6022012", Description: "提供Arm架构软件定义的存储服务，管理每节点不超过160TB裸容量磁盘，按照节点数目购买。", Module: "基础模块", Arch: "Arm", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "A", Unit: "每节点"},
		{ID: "ecf-arm-sdn-adv", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "Arm SDN高级网络服务云产品永久许可", Code: "11EP6004013", Description: "Arm SDN高级网络服务云产品，按套收费。", Module: "基础模块", Arch: "Arm", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "B1", Unit: "每套"},
		// ===== 升级产品 =====
		{ID: "ecf-x86-upgrade-v5", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "ECS V5升级ECF602许可（x86）永久许可-每节点", Code: "11EF6009UP3", Description: "ECS V5升级ECF602，按节点数购买。", Module: "基础模块", Arch: "X86", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "升级", Unit: "每节点"},
		{ID: "ecf-x86-upgrade-602", Category: "云基础设施ECF解决方案", Series: "易捷行云云基础设施平台V6", Name: "ECF V602升级ECF V611许可（x86）永久许可-每CPU", Code: "11EF6009UP4", Description: "ECF V602到ECF V611升级许可，按处理器购买。", Module: "基础模块", Arch: "X86", Product: "ECF V611", LicenseType: "正式（软件永久许可）", TypeClass: "升级", Unit: "每CPU"},
	}
}

// getServiceCatalog returns the full service catalog
func getServiceCatalog() []WBSService {
	return []WBSService{
		// ===== 安装部署服务 =====
		{ID: "svc-install-x86-10", Category: "安装部署服务", Name: "A类产品-x86 标准安装服务包-10节点", Code: "11ST6101001", Description: "约定的单一安装地点的易捷行云x86云产品A类产品一次初始化安装实施（限10节点含以内），测试，基本功能使用培训。", Unit: "套"},
		{ID: "svc-install-x86-20", Category: "安装部署服务", Name: "A类产品-x86 标准安装服务包-20节点", Code: "11ST6101002", Description: "约定的单一安装地点的易捷行云x86云产品A类产品一次初始化安装实施（限20节点含以内），测试，基本功能使用培训。", Unit: "套"},
		{ID: "svc-install-x86-50", Category: "安装部署服务", Name: "A类产品-x86 标准安装服务包-50节点", Code: "11ST6101003", Description: "约定的单一安装地点的易捷行云x86云产品A类产品一次初始化安装实施（限50节点含以内），测试，基本功能使用培训。", Unit: "套"},
		{ID: "svc-install-arm-10", Category: "安装部署服务", Name: "A类产品-Arm 标准安装服务包-10节点", Code: "11ST6101011", Description: "约定的单一安装地点的易捷行云Arm云产品A类产品一次初始化安装实施（限10节点含以内），测试，基本功能使用培训。", Unit: "套"},
		{ID: "svc-expand-x86-5", Category: "安装部署服务", Name: "A类产品-x86 标准扩容服务包-5节点", Code: "11ST6101021", Description: "已购环境节点扩容服务。", Unit: "套"},
		{ID: "svc-expand-arm-5", Category: "安装部署服务", Name: "A类产品-Arm 标准扩容服务包-5节点", Code: "11ST6101022", Description: "已购环境节点扩容服务。", Unit: "套"},
		// ===== 标准维保 =====
		{ID: "svc-maint-a-724", Category: "标准维保服务", Name: "A类云产品-1年7x24标准维保服务包（硬件≤5年）", Code: "11ST6102001", Description: "提供1年7*24小时服务响应，提供公司定义的产品标准维保服务。", Unit: "年"},
		{ID: "svc-maint-b-724", Category: "标准维保服务", Name: "B类云产品-1年7x24标准维保服务包（硬件≤5年）", Code: "11ST6102002", Description: "提供1年7*24小时服务响应，B类产品维保服务包购买前需先购买A类产品维保服务包。", Unit: "年"},
		{ID: "svc-maint-a-node-x86", Category: "标准维保服务", Name: "A类云产品-x86 1年7x24标准维保服务包-每节点（硬件≤5年）", Code: "11ST6102003", Description: "产品部分非标准折扣体系报价，所有A类节点数量总和统计，按节点购买。", Unit: "每节点/年"},
		{ID: "svc-maint-a-node-arm", Category: "标准维保服务", Name: "A类云产品-Arm 1年7x24标准维保服务包-每节点（硬件≤5年）", Code: "11ST6102004", Description: "产品部分非标准折扣体系报价，所有A类节点数量总和统计，按节点购买。", Unit: "每节点/年"},
		{ID: "svc-maint-b-product", Category: "标准维保服务", Name: "B类云产品-1年7x24标准维保服务包-每云产品（硬件≤5年）", Code: "11ST6102005", Description: "产品部分非标准折扣体系报价，所有B类产品数量总和统计，按年购买。", Unit: "每产品/年"},
		{ID: "svc-maint-upgrade", Category: "标准维保服务", Name: "A类云产品维保升级标准服务包-含1年7x24标准维保+1年EOS升级订阅", Code: "11ST6102006", Description: "提供1年7*24小时服务响应+标准维保+1年EOS大版本升级能力。", Unit: "年"},
		// ===== EOS升级订阅 =====
		{ID: "svc-eos-upgrade", Category: "EOS升级订阅", Name: "数字原生引擎EOS版本升级订阅服务包", Code: "11ST6103001", Description: "提供1年EOS大版本升级能力，按年购买。", Unit: "年"},
		// ===== 增值运维 =====
		{ID: "svc-ops-s1-20", Category: "增值运维服务", Name: "增值运维-S1-20节点", Code: "11ST6104001", Description: "提供ECF/ECNF（限20节点含以内）公司定义的增值运维-S1服务。", Unit: "套"},
		{ID: "svc-ops-s1-50", Category: "增值运维服务", Name: "增值运维-S1-50节点", Code: "11ST6104002", Description: "提供ECF/ECNF（限50节点含以内）公司定义的增值运维-S1服务。", Unit: "套"},
		{ID: "svc-ops-s2-20", Category: "增值运维服务", Name: "增值运维-S2-20节点", Code: "11ST6104003", Description: "提供ECF/ECNF（限20节点含以内）公司定义的增值运维-S2服务。", Unit: "套"},
		{ID: "svc-ops-s2-50", Category: "增值运维服务", Name: "增值运维-S2-50节点", Code: "11ST6104004", Description: "提供ECF/ECNF（限50节点含以内）公司定义的增值运维-S2服务。", Unit: "套"},
		{ID: "svc-ops-s3-20", Category: "增值运维服务", Name: "增值运维-S3-20节点", Code: "11ST6104005", Description: "提供ECF/ECNF（限20节点含以内）公司定义的增值运维-S3服务。", Unit: "套"},
		{ID: "svc-ops-s3-50", Category: "增值运维服务", Name: "增值运维-S3-50节点", Code: "11ST6104006", Description: "提供ECF/ECNF（限50节点含以内）公司定义的增值运维-S3服务。", Unit: "套"},
		{ID: "svc-ops-custom", Category: "增值运维服务", Name: "增值运维定制", Code: "11ST6104007", Description: "根据客户需求定制的增值运维服务，Case by case报价。", Unit: "人天"},
		// ===== 产品高级服务 =====
		{ID: "svc-adv-cloud", Category: "产品高级服务", Name: "高级云交付服务", Code: "11ST6105001", Description: "大规模部署的客户（50节点以上）选购，提供专业的工程师进行云环境规划设计、部署及项目管理服务。", Unit: "套"},
		{ID: "svc-adv-log", Category: "产品高级服务", Name: "日志审计及分析", Code: "11ST6105002", Description: "提供云平台日志审计、日志审计收集分析、告警、展示等功能。Case by case报价。", Unit: "套"},
		{ID: "svc-adv-inspect-remote", Category: "产品高级服务", Name: "季度远程巡检服务", Code: "11ST6105003", Description: "提供一年4次远程接入平台健康度检测，平台检查等服务，并提供巡检报告。", Unit: "年"},
		{ID: "svc-adv-inspect-onsite", Category: "产品高级服务", Name: "季度现场巡检服务", Code: "11ST6105004", Description: "提供一年4次现场接入平台健康度检测，平台检查等服务，并提供巡检报告。", Unit: "年"},
		{ID: "svc-adv-resource", Category: "产品高级服务", Name: "资源运营分析", Code: "11ST6105005", Description: "提供ECF/ECNF平台资源使用及资源运营建议服务，并提供资源运营分析报告。", Unit: "套"},
		{ID: "svc-adv-guard", Category: "产品高级服务", Name: "云上保驾护航服务", Code: "11ST6105006", Description: "提供重保期间或重大节假日期间的平台可用性现场保障服务。", Unit: "套"},
		{ID: "svc-adv-antifragile", Category: "产品高级服务", Name: "平台抗脆弱性验证服务", Code: "11ST6105007", Description: "提供支持客户进行应急演练平台抗脆弱性开关机重启的保障支持（按环境计算）。", Unit: "每环境"},
		{ID: "svc-adv-security", Category: "产品高级服务", Name: "等保安全测评的加固", Code: "11ST6105008", Description: "提供等保及安全测评的云平台加固评估及可加固内容变更支持服务。", Unit: "套"},
		{ID: "svc-adv-img-std", Category: "产品高级服务", Name: "标准VM镜像制作（单个）", Code: "11ST6105009", Description: "提供镜像库标准镜像模板的制作。", Unit: "个"},
		{ID: "svc-adv-img-custom", Category: "产品高级服务", Name: "定制化VM镜像制作（单个）", Code: "11ST6105010", Description: "提供镜像定制分区和修改操作系统软件包基线服务。", Unit: "个"},
		{ID: "svc-adv-logo", Category: "产品高级服务", Name: "更换ECF软件Logo（每套）", Code: "11ST6105011", Description: "提供更换ECF/ECNF LOGO图片和公司文字服务。", Unit: "每套"},
		{ID: "svc-adv-checklist", Category: "产品高级服务", Name: "环境检查核对的服务项", Code: "11ST6105012", Description: "提供云平台交付前核对Checklist服务（按环境计算）。", Unit: "每环境"},
		{ID: "svc-adv-docking", Category: "产品高级服务", Name: "对接高级服务", Code: "11ST6105013", Description: "包括对接评估、对接完整操作手册、以及对接之后的完整功能及性能测试报告。", Unit: "套"},
		{ID: "svc-adv-crypto", Category: "产品高级服务", Name: "云密评服务", Code: "11ST6105014", Description: "包括密评方案设计服务、密评建设整改服务及方案评审服务。", Unit: "套"},
		// ===== 服务人天 =====
		{ID: "svc-manday-remote", Category: "服务人天", Name: "远程服务人天", Code: "11ST6106001", Description: "易捷行云工程师的远程服务，最小消耗单位为1人天。", Unit: "人天"},
		{ID: "svc-manday-onsite", Category: "服务人天", Name: "现场服务人天", Code: "11ST6106002", Description: "易捷行云工程师的现场服务，最小消耗单位为1人天。", Unit: "人天"},
		{ID: "svc-manday-senior", Category: "服务人天", Name: "高级工程师支持服务人天", Code: "11ST6106003", Description: "易捷行云高级工程师的现场服务，最小消耗单位为1人天。", Unit: "人天"},
		{ID: "svc-manday-rd", Category: "服务人天", Name: "远程研发专家支持服务人天", Code: "11ST6106004", Description: "易捷行云研发专家远程服务，最小消耗单位为1人天。", Unit: "人天"},
		// ===== 培训服务 =====
		{ID: "svc-train-cka", Category: "培训服务", Name: "CKA培训及考试-每人次", Code: "11ST6107001", Description: "5天培训课程+认证考试。", Unit: "人次"},
		{ID: "svc-train-coa", Category: "培训服务", Name: "COA培训及考试-每人次", Code: "11ST6107002", Description: "5天培训课程+认证考试。", Unit: "人次"},
		{ID: "svc-train-arch", Category: "培训服务", Name: "ECF认证培训-云架构师-每人次", Code: "11ST6107003", Description: "8人以上集中授课，培训课程+认证考试共5天。", Unit: "人次"},
		{ID: "svc-train-engineer", Category: "培训服务", Name: "ECF认证培训-云服务工程师-每人次", Code: "11ST6107004", Description: "8人以上集中授课，培训课程+认证考试共3天。", Unit: "人次"},
		{ID: "svc-train-assist", Category: "培训服务", Name: "ECF认证培训-云助理工程师-每次", Code: "11ST6107005", Description: "不限人数单次远程授课，培训课程+认证考试共2天。", Unit: "次"},
	}
}

// =================== API Handlers ===================

// GetCatalog returns the full product & service catalog
func (h *WBSHandler) GetCatalog(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"products": getProductCatalog(),
			"services": getServiceCatalog(),
		},
	})
}

// WBSOrderItem represents an item in the order
type WBSOrderItem struct {
	ItemID   string `json:"item_id"`
	Name     string `json:"name"`
	Code     string `json:"code"`
	Quantity int    `json:"quantity"`
	Unit     string `json:"unit"`
	Category string `json:"category"`
	Arch     string `json:"arch,omitempty"`
}

// WBSOpportunity holds the business opportunity info
type WBSOpportunity struct {
	OpportunityName  string `json:"opportunity_name"`
	OpportunityNo    string `json:"opportunity_no"`
	SalesOrder       string `json:"sales_order"`
	ContractNo       string `json:"contract_no"`
	CustomerName     string `json:"customer_name"`
	Agent            string `json:"agent"`
	DeployLocation   string `json:"deploy_location"`
	SalesDirector    string `json:"sales_director"`
	SalesVP          string `json:"sales_vp"`
	Sales            string `json:"sales"`
	PreSales         string `json:"pre_sales"`
	DeliveryLeader   string `json:"delivery_leader_email"`
	ProjectManager   string `json:"project_manager_email"`
}

// WBSEnvironment represents deployment environment config
type WBSEnvironment struct {
	EnvName       string `json:"env_name"`       // 环境名称
	EnvType       string `json:"env_type"`       // 新建/扩容/纯服务/升级
	Product       string `json:"product"`        // ECF V611 / ECNF V611
	LicenseType   string `json:"license_type"`   // 正式/按需/预交付/POC
	Arch          string `json:"arch"`           // X86/Arm
	MaintenanceYr int    `json:"maintenance_yr"` // 维保年限
	SLA           string `json:"sla"`            // 7x24 / 5x9
	ChangeLogo    bool   `json:"change_logo"`    // 是否更换logo
}

// WBSSaveRequest is the full WBS order data
type WBSSaveRequest struct {
	Opportunity  WBSOpportunity   `json:"opportunity"`
	Environments []WBSEnvironment `json:"environments"`
	Products     []WBSOrderItem   `json:"products"`
	Services     []WBSOrderItem   `json:"services"`
	Remarks      string           `json:"remarks"`
}

// SaveOrder saves a WBS order and returns the summary
func (h *WBSHandler) SaveOrder(c *gin.Context) {
	var req WBSSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误: " + err.Error()})
		return
	}

	username, _ := c.Get("username")
	userID, _ := c.Get("user_id")

	// Build order summary
	var orderItems []model.WBSOrderItem
	for _, p := range req.Products {
		if p.Quantity > 0 {
			orderItems = append(orderItems, model.WBSOrderItem{
				ItemType: "product",
				ItemID:   p.ItemID,
				Name:     p.Name,
				Code:     p.Code,
				Quantity: p.Quantity,
				Unit:     p.Unit,
				Category: p.Category,
				Arch:     p.Arch,
			})
		}
	}
	for _, s := range req.Services {
		if s.Quantity > 0 {
			orderItems = append(orderItems, model.WBSOrderItem{
				ItemType: "service",
				ItemID:   s.ItemID,
				Name:     s.Name,
				Code:     s.Code,
				Quantity: s.Quantity,
				Unit:     s.Unit,
				Category: s.Category,
			})
		}
	}

	// Save to DB
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

	// Save order items
	for i := range orderItems {
		orderItems[i].OrderID = order.ID
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
			"order_id": order.ID,
			"items":    orderItems,
			"message":  "WBS订单保存成功",
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

// GetOrder returns a specific order with items
func (h *WBSHandler) GetOrder(c *gin.Context) {
	id := c.Param("id")
	var order model.WBSOrder
	if err := repository.DB.First(&order, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "订单不存在"})
		return
	}

	var items []model.WBSOrderItem
	repository.DB.Where("order_id = ?", order.ID).Find(&items)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"order": order,
			"items": items,
		},
	})
}

// ExportExcel generates and returns an Excel file for the WBS order
// following the standard WBS template structure with all sheets
func (h *WBSHandler) ExportExcel(c *gin.Context) {
	id := c.Param("id")
	var order model.WBSOrder
	if err := repository.DB.First(&order, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "订单不存在"})
		return
	}

	var items []model.WBSOrderItem
	repository.DB.Where("order_id = ?", order.ID).Find(&items)

	// Separate products and services
	var productItems, serviceItems []model.WBSOrderItem
	for _, item := range items {
		if item.ItemType == "product" {
			productItems = append(productItems, item)
		} else {
			serviceItems = append(serviceItems, item)
		}
	}

	// Generate Excel matching template structure
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

	// ========== Sheet 1: 0-商机 ==========
	sheet0 := "0-商机"
	f.SetSheetName("Sheet1", sheet0)
	f.SetColWidth(sheet0, "A", "A", 22)
	f.SetColWidth(sheet0, "B", "B", 40)
	f.SetColWidth(sheet0, "C", "C", 40)

	f.SetCellValue(sheet0, "A1", "基础信息(售前填写）")
	f.SetCellStyle(sheet0, "A1", "A1", titleStyle)
	f.SetCellValue(sheet0, "A2", "客户信息")
	f.SetCellStyle(sheet0, "A2", "A2", labelStyle)

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
		row := i + 3
		f.SetCellValue(sheet0, fmt.Sprintf("A%d", row), field.label)
		f.SetCellStyle(sheet0, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), labelStyle)
		f.SetCellValue(sheet0, fmt.Sprintf("B%d", row), field.value)
		f.SetCellStyle(sheet0, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), valueStyle)
		if field.hint != "" {
			f.SetCellValue(sheet0, fmt.Sprintf("C%d", row), field.hint)
			f.SetCellStyle(sheet0, fmt.Sprintf("C%d", row), fmt.Sprintf("C%d", row), hintStyle)
		}
	}

	// ========== Sheet 2: 4-order页面(自有产品汇总) ==========
	sheetProd := "4-order页面(自有产品汇总)"
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
	f.SetColWidth(sheetProd, "I", "I", 12)
	f.SetColWidth(sheetProd, "J", "J", 18)
	f.SetColWidth(sheetProd, "K", "K", 10)

	prodCatalog := getProductCatalog()
	row := 2
	for _, item := range productItems {
		// Find catalog entry for full details
		var catEntry *WBSProduct
		for idx := range prodCatalog {
			if prodCatalog[idx].ID == item.ItemID {
				catEntry = &prodCatalog[idx]
				break
			}
		}
		f.SetCellValue(sheetProd, fmt.Sprintf("A%d", row), item.Category)
		if catEntry != nil {
			f.SetCellValue(sheetProd, fmt.Sprintf("B%d", row), catEntry.Series)
		}
		f.SetCellValue(sheetProd, fmt.Sprintf("C%d", row), item.Name)
		f.SetCellValue(sheetProd, fmt.Sprintf("D%d", row), item.Code)
		f.SetCellValue(sheetProd, fmt.Sprintf("E%d", row), item.Quantity)
		if catEntry != nil {
			f.SetCellValue(sheetProd, fmt.Sprintf("F%d", row), catEntry.Description)
			f.SetCellValue(sheetProd, fmt.Sprintf("G%d", row), catEntry.Module)
			f.SetCellValue(sheetProd, fmt.Sprintf("H%d", row), catEntry.Arch)
			f.SetCellValue(sheetProd, fmt.Sprintf("I%d", row), catEntry.Product)
			f.SetCellValue(sheetProd, fmt.Sprintf("J%d", row), catEntry.LicenseType)
			f.SetCellValue(sheetProd, fmt.Sprintf("K%d", row), catEntry.TypeClass)
		} else {
			f.SetCellValue(sheetProd, fmt.Sprintf("H%d", row), item.Arch)
		}
		// Apply data style
		for col := 'A'; col <= 'K'; col++ {
			f.SetCellStyle(sheetProd, fmt.Sprintf("%c%d", col, row), fmt.Sprintf("%c%d", col, row), dataStyle)
		}
		row++
	}

	// ========== Sheet 3: 服务汇总 ==========
	sheetSvc := "服务汇总"
	f.NewSheet(sheetSvc)
	svcHeaders := []string{"服务类别", "服务名称", "服务编码", "数量", "单位", "服务说明"}
	for i, hdr := range svcHeaders {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetSvc, cell, hdr)
		f.SetCellStyle(sheetSvc, cell, cell, headerStyle)
	}
	f.SetColWidth(sheetSvc, "A", "A", 16)
	f.SetColWidth(sheetSvc, "B", "B", 45)
	f.SetColWidth(sheetSvc, "C", "C", 14)
	f.SetColWidth(sheetSvc, "D", "D", 8)
	f.SetColWidth(sheetSvc, "E", "E", 10)
	f.SetColWidth(sheetSvc, "F", "F", 60)

	svcCatalog := getServiceCatalog()
	row = 2
	for _, item := range serviceItems {
		var catEntry *WBSService
		for idx := range svcCatalog {
			if svcCatalog[idx].ID == item.ItemID {
				catEntry = &svcCatalog[idx]
				break
			}
		}
		f.SetCellValue(sheetSvc, fmt.Sprintf("A%d", row), item.Category)
		f.SetCellValue(sheetSvc, fmt.Sprintf("B%d", row), item.Name)
		f.SetCellValue(sheetSvc, fmt.Sprintf("C%d", row), item.Code)
		f.SetCellValue(sheetSvc, fmt.Sprintf("D%d", row), item.Quantity)
		f.SetCellValue(sheetSvc, fmt.Sprintf("E%d", row), item.Unit)
		if catEntry != nil {
			f.SetCellValue(sheetSvc, fmt.Sprintf("F%d", row), catEntry.Description)
		}
		for col := 'A'; col <= 'F'; col++ {
			f.SetCellStyle(sheetSvc, fmt.Sprintf("%c%d", col, row), fmt.Sprintf("%c%d", col, row), dataStyle)
		}
		row++
	}

	// ========== Sheet 4: order汇总（报价单核对页面） ==========
	sheetOrder := "order汇总（报价单核对页面）"
	f.NewSheet(sheetOrder)
	orderHeaders := []string{"序号", "类型", "产品大类/服务类别", "名称", "编码", "数量", "单位", "架构", "备注"}
	for i, hdr := range orderHeaders {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetOrder, cell, hdr)
		f.SetCellStyle(sheetOrder, cell, cell, headerStyle)
	}
	f.SetColWidth(sheetOrder, "A", "A", 6)
	f.SetColWidth(sheetOrder, "B", "B", 8)
	f.SetColWidth(sheetOrder, "C", "C", 22)
	f.SetColWidth(sheetOrder, "D", "D", 45)
	f.SetColWidth(sheetOrder, "E", "E", 14)
	f.SetColWidth(sheetOrder, "F", "F", 8)
	f.SetColWidth(sheetOrder, "G", "G", 10)
	f.SetColWidth(sheetOrder, "H", "H", 8)
	f.SetColWidth(sheetOrder, "I", "I", 20)

	row = 2
	seq := 1
	for _, item := range items {
		itemType := "产品"
		if item.ItemType == "service" {
			itemType = "服务"
		}
		f.SetCellValue(sheetOrder, fmt.Sprintf("A%d", row), seq)
		f.SetCellValue(sheetOrder, fmt.Sprintf("B%d", row), itemType)
		f.SetCellValue(sheetOrder, fmt.Sprintf("C%d", row), item.Category)
		f.SetCellValue(sheetOrder, fmt.Sprintf("D%d", row), item.Name)
		f.SetCellValue(sheetOrder, fmt.Sprintf("E%d", row), item.Code)
		f.SetCellValue(sheetOrder, fmt.Sprintf("F%d", row), item.Quantity)
		f.SetCellValue(sheetOrder, fmt.Sprintf("G%d", row), item.Unit)
		f.SetCellValue(sheetOrder, fmt.Sprintf("H%d", row), item.Arch)
		for col := 'A'; col <= 'I'; col++ {
			f.SetCellStyle(sheetOrder, fmt.Sprintf("%c%d", col, row), fmt.Sprintf("%c%d", col, row), dataStyle)
		}
		row++
		seq++
	}

	// ========== Sheet 5: 6-order页面(立项信息) ==========
	sheetProject := "6-order页面(立项信息)"
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

	// Environment info header
	f.SetCellValue(sheetProject, "A5", "环境信息")
	f.SetCellStyle(sheetProject, "A5", "A5", titleStyle)
	envHeaders := []string{"环境名称", "状态", "购买产品", "维保年限", "SLA", "license授权类型", "license架构类型"}
	for i, hdr := range envHeaders {
		cell := fmt.Sprintf("%c6", 'A'+i)
		f.SetCellValue(sheetProject, cell, hdr)
		f.SetCellStyle(sheetProject, cell, cell, headerStyle)
	}
	// Default env row
	f.SetCellValue(sheetProject, "A7", "第1套环境")
	f.SetCellValue(sheetProject, "B7", "新建")
	f.SetCellValue(sheetProject, "C7", "ECF V611")
	f.SetCellValue(sheetProject, "D7", "1")
	f.SetCellValue(sheetProject, "E7", "7x24")
	f.SetCellValue(sheetProject, "F7", "正式（软件永久许可）")
	f.SetCellValue(sheetProject, "G7", "X86")
	for col := 'A'; col <= 'G'; col++ {
		f.SetCellStyle(sheetProject, fmt.Sprintf("%c7", col), fmt.Sprintf("%c7", col), dataStyle)
	}

	// ========== Sheet 6: 5-order页面(自有产品按环境) ==========
	sheetProdEnv := "5-order页面(自有产品按环境）"
	f.NewSheet(sheetProdEnv)
	f.SetCellValue(sheetProdEnv, "A1", "第1套环境")
	f.SetCellStyle(sheetProdEnv, "A1", "A1", titleStyle)

	envOrderHeaders := []string{"状态", "购买产品", "维保年限", "SLA", "license授权类型", "license架构类型"}
	for i, hdr := range envOrderHeaders {
		cell := fmt.Sprintf("%c2", 'A'+i)
		f.SetCellValue(sheetProdEnv, cell, hdr)
		f.SetCellStyle(sheetProdEnv, cell, cell, headerStyle)
	}
	f.SetCellValue(sheetProdEnv, "A3", "新建")
	f.SetCellValue(sheetProdEnv, "B3", "ECF V611")
	f.SetCellValue(sheetProdEnv, "C3", "1")
	f.SetCellValue(sheetProdEnv, "D3", "7x24")
	f.SetCellValue(sheetProdEnv, "E3", "正式（软件永久许可）")
	f.SetCellValue(sheetProdEnv, "F3", "X86")
	for col := 'A'; col <= 'F'; col++ {
		f.SetCellStyle(sheetProdEnv, fmt.Sprintf("%c3", col), fmt.Sprintf("%c3", col), dataStyle)
	}

	prodEnvHeaders := []string{"产品大类", "产品系列", "产品名称", "产品编码", "数量", "产品说明"}
	for i, hdr := range prodEnvHeaders {
		cell := fmt.Sprintf("%c5", 'A'+i)
		f.SetCellValue(sheetProdEnv, cell, hdr)
		f.SetCellStyle(sheetProdEnv, cell, cell, headerStyle)
	}
	f.SetColWidth(sheetProdEnv, "A", "A", 20)
	f.SetColWidth(sheetProdEnv, "B", "B", 28)
	f.SetColWidth(sheetProdEnv, "C", "C", 45)
	f.SetColWidth(sheetProdEnv, "D", "D", 14)
	f.SetColWidth(sheetProdEnv, "E", "E", 8)
	f.SetColWidth(sheetProdEnv, "F", "F", 50)

	row = 6
	for _, item := range productItems {
		var catEntry *WBSProduct
		for idx := range prodCatalog {
			if prodCatalog[idx].ID == item.ItemID {
				catEntry = &prodCatalog[idx]
				break
			}
		}
		f.SetCellValue(sheetProdEnv, fmt.Sprintf("A%d", row), item.Category)
		if catEntry != nil {
			f.SetCellValue(sheetProdEnv, fmt.Sprintf("B%d", row), catEntry.Series)
		}
		f.SetCellValue(sheetProdEnv, fmt.Sprintf("C%d", row), item.Name)
		f.SetCellValue(sheetProdEnv, fmt.Sprintf("D%d", row), item.Code)
		f.SetCellValue(sheetProdEnv, fmt.Sprintf("E%d", row), item.Quantity)
		if catEntry != nil {
			f.SetCellValue(sheetProdEnv, fmt.Sprintf("F%d", row), catEntry.Description)
		}
		for col := 'A'; col <= 'F'; col++ {
			f.SetCellStyle(sheetProdEnv, fmt.Sprintf("%c%d", col, row), fmt.Sprintf("%c%d", col, row), dataStyle)
		}
		row++
	}

	// ========== Sheet 7: order模板 ==========
	sheetTmpl := "order模板"
	f.NewSheet(sheetTmpl)
	f.SetCellValue(sheetTmpl, "A1", "第1套环境")
	f.SetCellStyle(sheetTmpl, "A1", "A1", titleStyle)

	tmplHeaders := []string{"状态", "购买产品", "维保年限", "SLA", "license授权类型", "license架构类型"}
	for i, hdr := range tmplHeaders {
		cell := fmt.Sprintf("%c2", 'A'+i)
		f.SetCellValue(sheetTmpl, cell, hdr)
		f.SetCellStyle(sheetTmpl, cell, cell, headerStyle)
	}
	f.SetCellValue(sheetTmpl, "A3", "新建")
	f.SetCellValue(sheetTmpl, "B3", "ECF V611")
	f.SetCellValue(sheetTmpl, "C3", "1")
	f.SetCellValue(sheetTmpl, "D3", "7x24")
	f.SetCellValue(sheetTmpl, "E3", "正式（软件永久许可）")
	f.SetCellValue(sheetTmpl, "F3", "X86")
	for col := 'A'; col <= 'F'; col++ {
		f.SetCellStyle(sheetTmpl, fmt.Sprintf("%c3", col), fmt.Sprintf("%c3", col), dataStyle)
	}

	tmplItemHeaders := []string{"产品大类", "产品系列", "产品名称", "产品编码", "数量", "产品说明"}
	for i, hdr := range tmplItemHeaders {
		cell := fmt.Sprintf("%c4", 'A'+i)
		f.SetCellValue(sheetTmpl, cell, hdr)
		f.SetCellStyle(sheetTmpl, cell, cell, headerStyle)
	}
	f.SetColWidth(sheetTmpl, "A", "A", 20)
	f.SetColWidth(sheetTmpl, "B", "B", 28)
	f.SetColWidth(sheetTmpl, "C", "C", 45)
	f.SetColWidth(sheetTmpl, "D", "D", 14)
	f.SetColWidth(sheetTmpl, "E", "E", 8)
	f.SetColWidth(sheetTmpl, "F", "F", 50)

	// ========== Sheet 8: 整体模板 ==========
	sheetOverall := "整体模板"
	f.NewSheet(sheetOverall)
	f.SetCellValue(sheetOverall, "A1", "通用")
	f.SetCellValue(sheetOverall, "B1", "请填写人自行核对硬件兼容性及规格信息")
	f.SetCellStyle(sheetOverall, "A1", "A1", labelStyle)
	f.SetColWidth(sheetOverall, "A", "A", 8)
	f.SetColWidth(sheetOverall, "B", "B", 50)

	// Drop-down reference data
	f.SetCellValue(sheetOverall, "U1", "购买产品")
	f.SetCellValue(sheetOverall, "U2", "ECF V611")
	f.SetCellValue(sheetOverall, "U3", "ECNF V611")
	f.SetCellValue(sheetOverall, "W1", "license授权类型")
	f.SetCellValue(sheetOverall, "W2", "正式（软件永久许可）")
	f.SetCellValue(sheetOverall, "W3", "正式（软件订阅）")
	f.SetCellValue(sheetOverall, "Y1", "license架构类型")
	f.SetCellValue(sheetOverall, "Y2", "X86")
	f.SetCellValue(sheetOverall, "Y3", "Arm")

	// Set active sheet to order summary
	idx, _ := f.GetSheetIndex(sheetOrder)
	f.SetActiveSheet(idx)

	// Write to response
	filename := fmt.Sprintf("WBS_%s_%s.xlsx", order.OpportunityNo, time.Now().Format("20060102"))
	filename = strings.ReplaceAll(filename, " ", "_")

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Transfer-Encoding", "binary")

	if err := f.Write(c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "生成Excel失败: " + err.Error()})
	}
}

// DeleteOrder deletes a WBS order
func (h *WBSHandler) DeleteOrder(c *gin.Context) {
	id := c.Param("id")
	var order model.WBSOrder
	if err := repository.DB.First(&order, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "订单不存在"})
		return
	}

	// Delete items first
	repository.DB.Where("order_id = ?", order.ID).Delete(&model.WBSOrderItem{})
	// Delete order
	repository.DB.Delete(&order)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}
