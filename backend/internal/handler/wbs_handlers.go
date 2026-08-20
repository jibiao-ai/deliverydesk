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
		CustomerName:    req.Opportunity.CustomerName,
		Agent:           req.Opportunity.Agent,
		DeployLocation:  req.Opportunity.DeployLocation,
		Sales:           req.Opportunity.Sales,
		PreSales:        req.Opportunity.PreSales,
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
func (h *WBSHandler) ExportExcel(c *gin.Context) {
	id := c.Param("id")
	var order model.WBSOrder
	if err := repository.DB.First(&order, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "订单不存在"})
		return
	}

	var items []model.WBSOrderItem
	repository.DB.Where("order_id = ?", order.ID).Find(&items)

	// Generate Excel
	f := excelize.NewFile()
	defer f.Close()

	// Sheet 1: 商机信息
	sheet1 := "商机信息"
	f.SetSheetName("Sheet1", sheet1)
	f.SetCellValue(sheet1, "A1", "基础信息")
	f.SetCellValue(sheet1, "A3", "商机名称")
	f.SetCellValue(sheet1, "B3", order.OpportunityName)
	f.SetCellValue(sheet1, "A4", "商机号")
	f.SetCellValue(sheet1, "B4", order.OpportunityNo)
	f.SetCellValue(sheet1, "A5", "客户名称")
	f.SetCellValue(sheet1, "B5", order.CustomerName)
	f.SetCellValue(sheet1, "A6", "代理商")
	f.SetCellValue(sheet1, "B6", order.Agent)
	f.SetCellValue(sheet1, "A7", "部署地点")
	f.SetCellValue(sheet1, "B7", order.DeployLocation)
	f.SetCellValue(sheet1, "A8", "销售")
	f.SetCellValue(sheet1, "B8", order.Sales)
	f.SetCellValue(sheet1, "A9", "售前")
	f.SetCellValue(sheet1, "B9", order.PreSales)
	f.SetCellValue(sheet1, "A10", "项目经理邮箱")
	f.SetCellValue(sheet1, "B10", order.ProjectManager)

	// Sheet 2: Order 汇总
	sheet2 := "Order汇总"
	f.NewSheet(sheet2)

	// Header
	headers := []string{"类型", "产品大类", "产品名称", "产品编码", "数量", "单位"}
	for i, h := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheet2, cell, h)
	}

	// Style the header
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E2EFDA"}, Pattern: 1},
	})
	f.SetCellStyle(sheet2, "A1", "F1", headerStyle)

	row := 2
	for _, item := range items {
		itemType := "产品"
		if item.ItemType == "service" {
			itemType = "服务"
		}
		f.SetCellValue(sheet2, fmt.Sprintf("A%d", row), itemType)
		f.SetCellValue(sheet2, fmt.Sprintf("B%d", row), item.Category)
		f.SetCellValue(sheet2, fmt.Sprintf("C%d", row), item.Name)
		f.SetCellValue(sheet2, fmt.Sprintf("D%d", row), item.Code)
		f.SetCellValue(sheet2, fmt.Sprintf("E%d", row), item.Quantity)
		f.SetCellValue(sheet2, fmt.Sprintf("F%d", row), item.Unit)
		row++
	}

	// Set column widths
	f.SetColWidth(sheet2, "A", "A", 8)
	f.SetColWidth(sheet2, "B", "B", 25)
	f.SetColWidth(sheet2, "C", "C", 45)
	f.SetColWidth(sheet2, "D", "D", 15)
	f.SetColWidth(sheet2, "E", "E", 8)
	f.SetColWidth(sheet2, "F", "F", 10)

	// Set active sheet
	idx, _ := f.GetSheetIndex(sheet2)
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
