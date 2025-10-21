package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/edgelink/backend/internal/domain"
	"github.com/edgelink/backend/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminHandler 管理员操作处理器
type AdminHandler struct {
	deviceRepo        repository.DeviceRepository
	virtualNetworkRepo repository.VirtualNetworkRepository
	alertRepo         repository.AlertRepository
	auditLogRepo      repository.AuditLogRepository
	sessionRepo       repository.SessionRepository
}

// NewAdminHandler 创建AdminHandler实例
func NewAdminHandler(
	deviceRepo repository.DeviceRepository,
	virtualNetworkRepo repository.VirtualNetworkRepository,
	alertRepo repository.AlertRepository,
	auditLogRepo repository.AuditLogRepository,
	sessionRepo repository.SessionRepository,
) *AdminHandler {
	return &AdminHandler{
		deviceRepo:         deviceRepo,
		virtualNetworkRepo: virtualNetworkRepo,
		alertRepo:          alertRepo,
		auditLogRepo:       auditLogRepo,
		sessionRepo:        sessionRepo,
	}
}

// GetDevices godoc
// @Summary      获取设备列表
// @Description  获取所有设备列表，支持过滤和分页
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        virtual_network_id  query    string  false  "虚拟网络ID"
// @Param        online             query    string  false  "在线状态过滤 (true/false)"
// @Param        platform           query    string  false  "平台过滤"
// @Param        limit              query    int     false  "返回数量限制"
// @Param        offset             query    int     false  "偏移量"
// @Success      200  {object}  DeviceListResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/admin/devices [get]
func (h *AdminHandler) GetDevices(c *gin.Context) {
	// 解析查询参数
	var filters struct {
		VirtualNetworkID *uuid.UUID
		Online           *bool
		Platform         *domain.Platform
		Limit            int
		Offset           int
	}

	// 虚拟网络ID过滤
	if vnIDStr := c.Query("virtual_network_id"); vnIDStr != "" {
		vnID, err := uuid.Parse(vnIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_virtual_network_id",
				Message: "virtual_network_id must be a valid UUID",
			})
			return
		}
		filters.VirtualNetworkID = &vnID
	}

	// 在线状态过滤
	if onlineStr := c.Query("online"); onlineStr != "" {
		online := onlineStr == "true"
		filters.Online = &online
	}

	// 平台过滤
	if platformStr := c.Query("platform"); platformStr != "" {
		platform := domain.Platform(platformStr)
		filters.Platform = &platform
	}

	// 分页参数
	filters.Limit, _ = strconv.Atoi(c.DefaultQuery("limit", "50"))
	filters.Offset, _ = strconv.Atoi(c.DefaultQuery("offset", "0"))

	// 查询设备
	var devicesSlice []domain.Device
	var err error

	if filters.VirtualNetworkID != nil {
		devicesSlice, err = h.deviceRepo.FindByVirtualNetwork(c.Request.Context(), *filters.VirtualNetworkID, filters.Online)
	} else {
		// 查询所有设备（需要在DeviceRepository中添加FindAll方法）
		// 这里先用一个临时方案
		devicesSlice, err = h.deviceRepo.FindByVirtualNetwork(c.Request.Context(), uuid.Nil, nil)
	}

	// 转换为指针切片
	devices := make([]*domain.Device, len(devicesSlice))
	for i := range devicesSlice {
		devices[i] = &devicesSlice[i]
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "query_failed",
			Message: err.Error(),
		})
		return
	}

	// 应用平台过滤
	if filters.Platform != nil {
		filtered := make([]*domain.Device, 0)
		for _, d := range devices {
			if d.Platform == *filters.Platform {
				filtered = append(filtered, d)
			}
		}
		devices = filtered
	}

	// 应用分页
	total := len(devices)
	if filters.Offset < len(devices) {
		devices = devices[filters.Offset:]
	} else {
		devices = []*domain.Device{}
	}
	if filters.Limit > 0 && len(devices) > filters.Limit {
		devices = devices[:filters.Limit]
	}

	c.JSON(http.StatusOK, DeviceListResponse{
		Devices: devices,
		Total:   total,
		Limit:   filters.Limit,
		Offset:  filters.Offset,
	})
}

// GetVirtualNetworks godoc
// @Summary      获取虚拟网络列表
// @Description  获取组织的所有虚拟网络
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        organization_id  query    string  false  "组织ID"
// @Success      200  {object}  VirtualNetworkListResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/admin/virtual-networks [get]
func (h *AdminHandler) GetVirtualNetworks(c *gin.Context) {
	// TODO: 从认证上下文中获取组织ID，目前使用查询参数
	var organizationID uuid.UUID
	if orgIDStr := c.Query("organization_id"); orgIDStr != "" {
		var err error
		organizationID, err = uuid.Parse(orgIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_organization_id",
				Message: "organization_id must be a valid UUID",
			})
			return
		}
	}

	networksSlice, err := h.virtualNetworkRepo.FindByOrganization(c.Request.Context(), organizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "query_failed",
			Message: err.Error(),
		})
		return
	}

	// 转换为指针切片
	networks := make([]*domain.VirtualNetwork, len(networksSlice))
	for i := range networksSlice {
		networks[i] = &networksSlice[i]
	}

	c.JSON(http.StatusOK, VirtualNetworkListResponse{
		Networks: networks,
		Total:    len(networks),
	})
}

// CreateVirtualNetwork godoc
// @Summary      创建虚拟网络
// @Description  为组织创建新的虚拟网络
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        request  body  CreateVirtualNetworkRequest  true  "创建请求"
// @Success      201  {object}  domain.VirtualNetwork
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/admin/virtual-networks [post]
func (h *AdminHandler) CreateVirtualNetwork(c *gin.Context) {
	var req CreateVirtualNetworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// 解析组织ID
	organizationID, err := uuid.Parse(req.OrganizationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_organization_id",
			Message: "organization_id must be a valid UUID",
		})
		return
	}

	// 创建虚拟网络
	network := &domain.VirtualNetwork{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		Name:           req.Name,
		CIDR:           req.CIDR,
		GatewayIP:      req.GatewayIP,
		DNSServers:     req.DNSServers,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := h.virtualNetworkRepo.Create(c.Request.Context(), network); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "creation_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, network)
}

// DeleteDevice godoc
// @Summary      删除/撤销设备
// @Description  从虚拟网络中撤销设备
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        device_id  path  string  true  "设备ID"
// @Success      200  {object}  SuccessResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/admin/devices/{device_id} [delete]
func (h *AdminHandler) DeleteDevice(c *gin.Context) {
	deviceIDStr := c.Param("device_id")
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_device_id",
			Message: "device_id must be a valid UUID",
		})
		return
	}

	// 检查设备是否存在
	device, err := h.deviceRepo.FindByID(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "device_not_found",
			Message: "device not found",
		})
		return
	}

	// 删除设备
	if err := h.deviceRepo.Delete(c.Request.Context(), device.ID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "deletion_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "device deleted successfully",
	})
}

// GetAlerts godoc
// @Summary      获取告警列表
// @Description  获取告警列表，支持过滤和分页
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        device_id   query    string  false  "设备ID"
// @Param        severity    query    string  false  "严重程度 (critical/high/medium/low)"
// @Param        status      query    string  false  "状态 (active/acknowledged/resolved)"
// @Param        type        query    string  false  "告警类型"
// @Param        limit       query    int     false  "返回数量限制"
// @Param        offset      query    int     false  "偏移量"
// @Success      200  {object}  AlertListResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/admin/alerts [get]
func (h *AdminHandler) GetAlerts(c *gin.Context) {
	filters := &repository.AlertFilters{
		Limit:  50,
		Offset: 0,
	}

	// 解析过滤参数
	if deviceIDStr := c.Query("device_id"); deviceIDStr != "" {
		deviceID, err := uuid.Parse(deviceIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_device_id",
				Message: "device_id must be a valid UUID",
			})
			return
		}
		filters.DeviceID = &deviceID
	}

	if severityStr := c.Query("severity"); severityStr != "" {
		severity := domain.Severity(severityStr)
		filters.Severity = &severity
	}

	if statusStr := c.Query("status"); statusStr != "" {
		status := domain.AlertStatus(statusStr)
		filters.Status = &status
	}

	if typeStr := c.Query("type"); typeStr != "" {
		alertType := domain.AlertType(typeStr)
		filters.AlertType = &alertType
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		filters.Limit, _ = strconv.Atoi(limitStr)
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		filters.Offset, _ = strconv.Atoi(offsetStr)
	}

	// 查询告警
	alerts, total, err := h.alertRepo.FindByFilters(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "query_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, AlertListResponse{
		Alerts: alerts,
		Total:  int(total),
		Limit:  filters.Limit,
		Offset: filters.Offset,
	})
}

// AcknowledgeAlert godoc
// @Summary      确认告警
// @Description  标记告警为已确认状态
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        alert_id  path  string  true  "告警ID"
// @Param        request   body  AcknowledgeAlertRequest  true  "确认请求"
// @Success      200  {object}  SuccessResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/admin/alerts/{alert_id}/acknowledge [post]
func (h *AdminHandler) AcknowledgeAlert(c *gin.Context) {
	alertIDStr := c.Param("alert_id")
	alertID, err := uuid.Parse(alertIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_alert_id",
			Message: "alert_id must be a valid UUID",
		})
		return
	}

	var req AcknowledgeAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// 解析确认者ID
	acknowledgedBy, err := uuid.Parse(req.AcknowledgedBy)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_acknowledged_by",
			Message: "acknowledged_by must be a valid UUID",
		})
		return
	}

	// 确认告警
	if err := h.alertRepo.Acknowledge(c.Request.Context(), alertID, acknowledgedBy); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "acknowledgement_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "alert acknowledged successfully",
	})
}

// GetAuditLogs godoc
// @Summary      获取审计日志
// @Description  获取审计日志列表，支持过滤和分页
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        organization_id  query    string  false  "组织ID"
// @Param        actor_id         query    string  false  "操作者ID"
// @Param        action           query    string  false  "操作类型"
// @Param        resource_type    query    string  false  "资源类型"
// @Param        resource_id      query    string  false  "资源ID"
// @Param        start_time       query    string  false  "开始时间 (RFC3339)"
// @Param        end_time         query    string  false  "结束时间 (RFC3339)"
// @Param        limit            query    int     false  "返回数量限制"
// @Param        offset           query    int     false  "偏移量"
// @Success      200  {object}  AuditLogListResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/admin/audit-logs [get]
func (h *AdminHandler) GetAuditLogs(c *gin.Context) {
	filters := &repository.AuditLogFilters{
		Limit:  50,
		Offset: 0,
	}

	// 解析过滤参数
	if orgIDStr := c.Query("organization_id"); orgIDStr != "" {
		orgID, err := uuid.Parse(orgIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_organization_id",
				Message: "organization_id must be a valid UUID",
			})
			return
		}
		filters.OrganizationID = &orgID
	}

	if actorIDStr := c.Query("actor_id"); actorIDStr != "" {
		actorID, err := uuid.Parse(actorIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_actor_id",
				Message: "actor_id must be a valid UUID",
			})
			return
		}
		filters.ActorID = &actorID
	}

	if action := c.Query("action"); action != "" {
		filters.Action = &action
	}

	if resourceTypeStr := c.Query("resource_type"); resourceTypeStr != "" {
		resourceType := domain.ResourceType(resourceTypeStr)
		filters.ResourceType = &resourceType
	}

	if resourceIDStr := c.Query("resource_id"); resourceIDStr != "" {
		resourceID, err := uuid.Parse(resourceIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_resource_id",
				Message: "resource_id must be a valid UUID",
			})
			return
		}
		filters.ResourceID = &resourceID
	}

	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_start_time",
				Message: "start_time must be in RFC3339 format",
			})
			return
		}
		filters.StartTime = &startTime
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		endTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_end_time",
				Message: "end_time must be in RFC3339 format",
			})
			return
		}
		filters.EndTime = &endTime
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		filters.Limit, _ = strconv.Atoi(limitStr)
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		filters.Offset, _ = strconv.Atoi(offsetStr)
	}

	// 查询审计日志
	logs, total, err := h.auditLogRepo.FindByFilters(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "query_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, AuditLogListResponse{
		Logs:   logs,
		Total:  int(total),
		Limit:  filters.Limit,
		Offset: filters.Offset,
	})
}

// 请求/响应类型定义

type DeviceListResponse struct {
	Devices []*domain.Device `json:"devices"`
	Total   int              `json:"total"`
	Limit   int              `json:"limit"`
	Offset  int              `json:"offset"`
}

type VirtualNetworkListResponse struct {
	Networks []*domain.VirtualNetwork `json:"networks"`
	Total    int                      `json:"total"`
}

type CreateVirtualNetworkRequest struct {
	OrganizationID string   `json:"organization_id" binding:"required"`
	Name           string   `json:"name" binding:"required"`
	CIDR           string   `json:"cidr" binding:"required"`
	GatewayIP      string   `json:"gateway_ip" binding:"required"`
	DNSServers     []string `json:"dns_servers"`
}

type AlertListResponse struct {
	Alerts []*domain.Alert `json:"alerts"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

type AcknowledgeAlertRequest struct {
	AcknowledgedBy string `json:"acknowledged_by" binding:"required"`
}

type AuditLogListResponse struct {
	Logs   []*domain.AuditLog `json:"logs"`
	Total  int                `json:"total"`
	Limit  int                `json:"limit"`
	Offset int                `json:"offset"`
}
