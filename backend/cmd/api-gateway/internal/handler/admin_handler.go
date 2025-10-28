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

// GetDeviceById godoc
// @Summary      获取设备详情
// @Description  获取设备的详细信息
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        device_id  path  string  true  "设备ID"
// @Success      200  {object}  DeviceDetailResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /api/v1/admin/devices/{device_id} [get]
func (h *AdminHandler) GetDeviceById(c *gin.Context) {
	deviceIDStr := c.Param("device_id")
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_device_id",
			Message: "device_id must be a valid UUID",
		})
		return
	}

	device, err := h.deviceRepo.FindByID(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "device_not_found",
			Message: "device not found",
		})
		return
	}

	// 获取设备的活跃会话
	sessions, err := h.sessionRepo.FindByDevice(c.Request.Context(), deviceID)
	if err != nil {
		// 如果获取会话失败，继续执行但会话数据为空
		sessions = []*domain.Session{}
	}

	// 获取设备的告警
	alertFilters := &repository.AlertFilters{
		DeviceID: &deviceID,
		Status:   &domain.AlertStatusActive,
		Limit:    10,
		Offset:   0,
	}
	alerts, _, err := h.alertRepo.FindByFilters(c.Request.Context(), alertFilters)
	if err != nil {
		alerts = []*domain.Alert{}
	}

	c.JSON(http.StatusOK, DeviceDetailResponse{
		Device:   device,
		Sessions: sessions,
		Alerts:   alerts,
	})
}

// GetDevicePeers godoc
// @Summary      获取设备对等列表
// @Description  获取设备的所有对等连接信息
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        device_id  path  string  true  "设备ID"
// @Success      200  {object}  DevicePeersResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /api/v1/admin/devices/{device_id}/peers [get]
func (h *AdminHandler) GetDevicePeers(c *gin.Context) {
	deviceIDStr := c.Param("device_id")
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_device_id",
			Message: "device_id must be a valid UUID",
		})
		return
	}

	// 获取设备活跃会话
	sessions, err := h.sessionRepo.FindByDevice(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "query_failed",
			Message: err.Error(),
		})
		return
	}

	// 构建对等设备响应
	peers := make([]*DevicePeer, 0)
	for _, session := range sessions {
		// 确定对等设备ID
		var peerDeviceID uuid.UUID
		if session.DeviceAID == deviceID {
			peerDeviceID = session.DeviceBID
		} else {
			peerDeviceID = session.DeviceAID
		}

		// 获取对等设备信息
		peerDevice, err := h.deviceRepo.FindByID(c.Request.Context(), peerDeviceID)
		if err != nil {
			continue // 跳过无法找到的对等设备
		}

		status := "connected"
		if session.EndedAt != nil {
			status = "disconnected"
		}

		peer := &DevicePeer{
			PeerID:           peerDevice.ID,
			PeerName:         peerDevice.Name,
			PeerIP:           peerDevice.VirtualIP,
			Status:           status,
			Latency:          session.LatencyMs,
			LastHandshake:    session.LastHandshakeAt,
			ConnectionType:   session.ConnectionType,
			BytesSent:        session.BytesSentA,
			BytesReceived:    session.BytesReceivedA,
		}

		// 如果当前设备是B，交换发送/接收字节
		if session.DeviceBID == deviceID {
			peer.BytesSent = session.BytesSentB
			peer.BytesReceived = session.BytesReceivedB
		}

		peers = append(peers, peer)
	}

	c.JSON(http.StatusOK, DevicePeersResponse{
		Peers: peers,
		Total: len(peers),
	})
}

// GetDeviceMetrics godoc
// @Summary      获取设备指标历史
// @Description  获取设备的历史指标数据
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        device_id  path  string  true  "设备ID"
// @Param        time_range query    string  false  "时间范围 (1h, 24h, 7d, 30d)"
// @Param        start_time query    string  false  "开始时间 (RFC3339)"
// @Param        end_time   query    string  false  "结束时间 (RFC3339)"
// @Success      200  {object}  DeviceMetricsResponse
// @Failure      400  {object}  ErrorResponse
// @Router       /api/v1/admin/devices/{device_id}/metrics [get]
func (h *AdminHandler) GetDeviceMetrics(c *gin.Context) {
	deviceIDStr := c.Param("device_id")
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_device_id",
			Message: "device_id must be a valid UUID",
		})
		return
	}

	// 解析时间范围
	timeRange := c.DefaultQuery("time_range", "24h")
	var endTime time.Time = time.Now()
	var startTime time.Time

	switch timeRange {
	case "1h":
		startTime = endTime.Add(-1 * time.Hour)
	case "24h":
		startTime = endTime.Add(-24 * time.Hour)
	case "7d":
		startTime = endTime.Add(-7 * 24 * time.Hour)
	case "30d":
		startTime = endTime.Add(-30 * 24 * time.Hour)
	default:
		// 自定义时间范围
		if startTimeStr := c.Query("start_time"); startTimeStr != "" {
			startTime, err = time.Parse(time.RFC3339, startTimeStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, ErrorResponse{
					Error:   "invalid_start_time",
					Message: "start_time must be in RFC3339 format",
				})
				return
			}
		} else {
			startTime = endTime.Add(-24 * time.Hour)
		}

		if endTimeStr := c.Query("end_time"); endTimeStr != "" {
			endTime, err = time.Parse(time.RFC3339, endTimeStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, ErrorResponse{
					Error:   "invalid_end_time",
					Message: "end_time must be in RFC3339 format",
				})
				return
			}
		}
	}

	// 获取设备的会话指标数据
	sessions, err := h.sessionRepo.FindByDeviceTimeRange(c.Request.Context(), deviceID, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "query_failed",
			Message: err.Error(),
		})
		return
	}

	// 构建指标数据点
	metrics := make([]*DeviceMetricsPoint, 0)
	for _, session := range sessions {
		metric := &DeviceMetricsPoint{
			Timestamp:     session.StartedAt,
			Latency:       session.LatencyMs,
			BytesSent:     session.BytesSentA,
			BytesReceived: session.BytesReceivedA,
		}

		// 如果当前设备是B，交换发送/接收字节
		if session.DeviceBID == deviceID {
			metric.BytesSent = session.BytesSentB
			metric.BytesReceived = session.BytesReceivedB
		}

		metrics = append(metrics, metric)
	}

	c.JSON(http.StatusOK, DeviceMetricsResponse{
		DeviceID:   deviceID,
		TimeRange:  timeRange,
		StartTime:  startTime,
		EndTime:    endTime,
		Metrics:    metrics,
		Total:      len(metrics),
	})
}

// GetDashboardStats godoc
// @Summary      获取仪表板统计数据
// @Description  获取仪表板所需的统计概览数据
// @Tags         stats
// @Accept       json
// @Produce      json
// @Param        organization_id  query    string  false  "组织ID"
// @Success      200  {object}  DashboardStatsResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/stats/dashboard [get]
func (h *AdminHandler) GetDashboardStats(c *gin.Context) {
	var organizationID *uuid.UUID
	if orgIDStr := c.Query("organization_id"); orgIDStr != "" {
		orgID, err := uuid.Parse(orgIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_organization_id",
				Message: "organization_id must be a valid UUID",
			})
			return
		}
		organizationID = &orgID
	}

	// 获取设备统计
	totalDevices, err := h.deviceRepo.CountByOrganization(c.Request.Context(), organizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "query_failed",
			Message: err.Error(),
		})
		return
	}

	onlineDevices, err := h.deviceRepo.CountByOrganization(c.Request.Context(), organizationID)
	if err != nil {
		onlineDevices = 0
	}

	// 获取告警统计
	activeAlerts, err := h.alertRepo.CountByStatus(c.Request.Context(), domain.AlertStatusActive)
	if err != nil {
		activeAlerts = 0
	}

	criticalAlerts, err := h.alertRepo.CountBySeverity(c.Request.Context(), domain.SeverityCritical)
	if err != nil {
		criticalAlerts = 0
	}

	// 获取活跃会话统计
	activeSessions, err := h.sessionRepo.CountActive(c.Request.Context())
	if err != nil {
		activeSessions = 0
	}

	// 模拟一些统计数据（实际应该从Prometheus或其他监控系统获取）
	tunnelSuccessRate := 85.6 // 85.6% P2P success rate
	avgLatency := 45.2        // 45.2ms average latency
	totalBandwidth := int64(1024 * 1024 * 500) // 500MB total bandwidth

	c.JSON(http.StatusOK, DashboardStatsResponse{
		DeviceStats: DeviceStats{
			Total:  totalDevices,
			Online: onlineDevices,
			Offline: totalDevices - onlineDevices,
		},
		AlertStats: AlertStats{
			Total:      activeAlerts,
			Critical:   criticalAlerts,
			High:       activeAlerts - criticalAlerts,
			Medium:     0,
			Low:        0,
		},
		SessionStats: SessionStats{
			Active:           activeSessions,
			SuccessRate:      tunnelSuccessRate,
			AverageLatency:   avgLatency,
			TotalBandwidth:   totalBandwidth,
		},
		UpdatedAt: time.Now(),
	})
}

// GetDeviceTrend godoc
// @Summary      获取设备趋势数据
// @Description  获取设备数量随时间变化的趋势数据
// @Tags         stats
// @Accept       json
// @Produce      json
// @Param        time_range  query    string  false  "时间范围 (1h, 24h, 7d, 30d)"
// @Param        organization_id query    string  false  "组织ID"
// @Success      200  {object}  DeviceTrendResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/stats/devices/trend [get]
func (h *AdminHandler) GetDeviceTrend(c *gin.Context) {
	timeRange := c.DefaultQuery("time_range", "24h")

	// 生成模拟的趋势数据
	var dataPoints []*DeviceTrendPoint
	now := time.Now()

	var interval time.Duration
	var points int

	switch timeRange {
	case "1h":
		interval = 5 * time.Minute
		points = 12
	case "24h":
		interval = 1 * time.Hour
		points = 24
	case "7d":
		interval = 6 * time.Hour
		points = 28
	case "30d":
		interval = 24 * time.Hour
		points = 30
	default:
		interval = 1 * time.Hour
		points = 24
	}

	for i := points - 1; i >= 0; i-- {
		timestamp := now.Add(-time.Duration(i) * interval)
		// 模拟数据：设备数量在 80-100 之间波动
		totalDevices := 80 + (i % 20)
		onlineDevices := int(float64(totalDevices) * (0.7 + 0.2*float64(i%5)/5))

		dataPoints = append(dataPoints, &DeviceTrendPoint{
			Timestamp:    timestamp,
			TotalDevices: totalDevices,
			OnlineDevices: onlineDevices,
		})
	}

	c.JSON(http.StatusOK, DeviceTrendResponse{
		TimeRange: timeRange,
		Data:      dataPoints,
		Total:     len(dataPoints),
	})
}

// GetTrafficStats godoc
// @Summary      获取流量统计数据
// @Description  获取网络流量统计信息
// @Tags         stats
// @Accept       json
// @Produce      json
// @Param        time_range  query    string  false  "时间范围 (1h, 24h, 7d, 30d)"
// @Success      200  {object}  TrafficStatsResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/stats/traffic [get]
func (h *AdminHandler) GetTrafficStats(c *gin.Context) {
	timeRange := c.DefaultQuery("time_range", "24h")

	// 生成模拟的流量数据
	var dataPoints []*TrafficPoint
	now := time.Now()

	var interval time.Duration
	var points int

	switch timeRange {
	case "1h":
		interval = 5 * time.Minute
		points = 12
	case "24h":
		interval = 1 * time.Hour
		points = 24
	case "7d":
		interval = 6 * time.Hour
		points = 28
	case "30d":
		interval = 24 * time.Hour
		points = 30
	default:
		interval = 1 * time.Hour
		points = 24
	}

	for i := points - 1; i >= 0; i-- {
		timestamp := now.Add(-time.Duration(i) * interval)
		// 模拟流量数据：在 100MB - 500MB 之间波动
		upload := 100 + (i % 400)
		download := 200 + (i % 300)

		dataPoints = append(dataPoints, &TrafficPoint{
			Timestamp: timestamp,
			Upload:   int64(upload * 1024 * 1024),   // Convert to bytes
			Download: int64(download * 1024 * 1024), // Convert to bytes
		})
	}

	c.JSON(http.StatusOK, TrafficStatsResponse{
		TimeRange: timeRange,
		Data:      dataPoints,
		Total:     len(dataPoints),
	})
}

// GetDeviceDistribution godoc
// @Summary      获取设备平台分布
// @Description  获取按平台分类的设备分布统计
// @Tags         stats
// @Accept       json
// @Produce      json
// @Param        organization_id query    string  false  "组织ID"
// @Success      200  {object}  DeviceDistributionResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/stats/devices/distribution [get]
func (h *AdminHandler) GetDeviceDistribution(c *gin.Context) {
	// 模拟设备分布数据
	distribution := []*PlatformDistribution{
		{
			Platform:  "desktop_linux",
			Count:     45,
			Percentage: 45.0,
		},
		{
			Platform:  "desktop_windows",
			Count:     30,
			Percentage: 30.0,
		},
		{
			Platform:  "desktop_macos",
			Count:     20,
			Percentage: 20.0,
		},
		{
			Platform:  "iot",
			Count:     5,
			Percentage: 5.0,
		},
	}

	c.JSON(http.StatusOK, DeviceDistributionResponse{
		Distribution: distribution,
		Total:        100,
	})
}

// GetAlertTrend godoc
// @Summary      获取告警趋势数据
// @Description  获取告警数量随时间变化的趋势数据
// @Tags         stats
// @Accept       json
// @Produce      json
// @Param        time_range  query    string  false  "时间范围 (1h, 24h, 7d, 30d)"
// @Success      200  {object}  AlertTrendResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/stats/alerts/trend [get]
func (h *AdminHandler) GetAlertTrend(c *gin.Context) {
	timeRange := c.DefaultQuery("time_range", "7d")

	// 生成模拟的告警趋势数据
	var dataPoints []*AlertTrendPoint
	now := time.Now()

	var interval time.Duration
	var points int

	switch timeRange {
	case "1h":
		interval = 5 * time.Minute
		points = 12
	case "24h":
		interval = 1 * time.Hour
		points = 24
	case "7d":
		interval = 6 * time.Hour
		points = 28
	case "30d":
		interval = 24 * time.Hour
		points = 30
	default:
		interval = 6 * time.Hour
		points = 28
	}

	for i := points - 1; i >= 0; i-- {
		timestamp := now.Add(-time.Duration(i) * interval)
		// 模拟告警数据：Critical: 0-2, High: 1-5, Medium: 2-8, Low: 3-10
		critical := i % 3
		high := 2 + (i % 4)
		medium := 3 + (i % 6)
		low := 5 + (i % 6)

		dataPoints = append(dataPoints, &AlertTrendPoint{
			Timestamp: timestamp,
			Critical:  critical,
			High:      high,
			Medium:    medium,
			Low:       low,
		})
	}

	c.JSON(http.StatusOK, AlertTrendResponse{
		TimeRange: timeRange,
		Data:      dataPoints,
		Total:     len(dataPoints),
	})
}

// GetTopologyDevices godoc
// @Summary      获取拓扑设备数据
// @Description  获取网络拓扑可视化所需的设备数据
// @Tags         topology
// @Accept       json
// @Produce      json
// @Param        organization_id query    string  false  "组织ID"
// @Success      200  {object}  TopologyDevicesResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/topology/devices [get]
func (h *AdminHandler) GetTopologyDevices(c *gin.Context) {
	// 获取所有设备
	var devices []*domain.Device
	var err error

	if orgIDStr := c.Query("organization_id"); orgIDStr != "" {
		orgID, err := uuid.Parse(orgIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_organization_id",
				Message: "organization_id must be a valid UUID",
			})
			return
		}
		// 这里需要在Repository中添加FindByOrganization方法
		// 暂时使用现有方法
		devicesSlice, err := h.deviceRepo.FindByVirtualNetwork(c.Request.Context(), uuid.Nil, nil)
		devices = make([]*domain.Device, len(devicesSlice))
		for i := range devicesSlice {
			devices[i] = &devicesSlice[i]
		}
	} else {
		devicesSlice, err := h.deviceRepo.FindByVirtualNetwork(c.Request.Context(), uuid.Nil, nil)
		devices = make([]*domain.Device, len(devicesSlice))
		for i := range devicesSlice {
			devices[i] = &devicesSlice[i]
		}
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "query_failed",
			Message: err.Error(),
		})
		return
	}

	// 转换为拓扑数据格式
	topologyDevices := make([]*TopologyDevice, 0)
	for _, device := range devices {
		topologyDevice := &TopologyDevice{
			ID:          device.ID,
			Name:        device.Name,
			VirtualIP:   device.VirtualIP,
			Platform:    string(device.Platform),
			NATType:     string(device.NATType),
			IsOnline:    device.Online,
			LastSeen:    device.LastSeenAt,
			VirtualNetworkID: device.VirtualNetworkID,
		}
		topologyDevices = append(topologyDevices, topologyDevice)
	}

	c.JSON(http.StatusOK, TopologyDevicesResponse{
		Devices: topologyDevices,
		Total:   len(topologyDevices),
	})
}

// GetTopologyPeers godoc
// @Summary      获取拓扑对等配置数据
// @Description  获取网络拓扑可视化所需的设备对等连接数据
// @Tags         topology
// @Accept       json
// @Produce      json
// @Param        organization_id query    string  false  "组织ID"
// @Success      200  {object}  TopologyPeersResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/topology/peers [get]
func (h *AdminHandler) GetTopologyPeers(c *gin.Context) {
	// 获取所有活跃会话作为对等连接
	sessions, err := h.sessionRepo.FindActive(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "query_failed",
			Message: err.Error(),
		})
		return
	}

	// 转换为拓扑数据格式
	peers := make([]*TopologyPeer, 0)
	for _, session := range sessions {
		peer := &TopologyPeer{
			ID:             session.ID,
			DeviceAID:      session.DeviceAID,
			DeviceBID:      session.DeviceBID,
			ConnectionType: string(session.ConnectionType),
			Latency:        session.LatencyMs,
			LastHandshake:  session.LastHandshakeAt,
			StartedAt:      session.StartedAt,
			EndpointA:      session.EndpointA,
			EndpointB:      session.EndpointB,
		}
		peers = append(peers, peer)
	}

	c.JSON(http.StatusOK, TopologyPeersResponse{
		Peers: peers,
		Total: len(peers),
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

// 设备详情响应
type DeviceDetailResponse struct {
	Device   *domain.Device     `json:"device"`
	Sessions []*domain.Session   `json:"sessions"`
	Alerts   []*domain.Alert     `json:"alerts"`
}

// 设备对等响应
type DevicePeersResponse struct {
	Peers []*DevicePeer `json:"peers"`
	Total int           `json:"total"`
}

type DevicePeer struct {
	PeerID           uuid.UUID `json:"peer_id"`
	PeerName         string    `json:"peer_name"`
	PeerIP           string    `json:"peer_ip"`
	Status           string    `json:"status"`
	Latency          *int      `json:"latency"`
	LastHandshake    *time.Time `json:"last_handshake"`
	ConnectionType   string    `json:"connection_type"`
	BytesSent        int64     `json:"bytes_sent"`
	BytesReceived    int64     `json:"bytes_received"`
}

// 设备指标响应
type DeviceMetricsResponse struct {
	DeviceID   uuid.UUID              `json:"device_id"`
	TimeRange  string                 `json:"time_range"`
	StartTime  time.Time              `json:"start_time"`
	EndTime    time.Time              `json:"end_time"`
	Metrics    []*DeviceMetricsPoint  `json:"metrics"`
	Total      int                    `json:"total"`
}

type DeviceMetricsPoint struct {
	Timestamp     time.Time `json:"timestamp"`
	Latency       *int      `json:"latency"`
	BytesSent     int64     `json:"bytes_sent"`
	BytesReceived int64     `json:"bytes_received"`
}

// 仪表板统计响应
type DashboardStatsResponse struct {
	DeviceStats  DeviceStats   `json:"device_stats"`
	AlertStats   AlertStats    `json:"alert_stats"`
	SessionStats SessionStats  `json:"session_stats"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

type DeviceStats struct {
	Total  int `json:"total"`
	Online int `json:"online"`
	Offline int `json:"offline"`
}

type AlertStats struct {
	Total    int `json:"total"`
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
}

type SessionStats struct {
	Active           int     `json:"active"`
	SuccessRate      float64 `json:"success_rate"`
	AverageLatency   float64 `json:"average_latency"`
	TotalBandwidth   int64   `json:"total_bandwidth"`
}

// 设备趋势响应
type DeviceTrendResponse struct {
	TimeRange string              `json:"time_range"`
	Data      []*DeviceTrendPoint `json:"data"`
	Total     int                 `json:"total"`
}

type DeviceTrendPoint struct {
	Timestamp     time.Time `json:"timestamp"`
	TotalDevices  int       `json:"total_devices"`
	OnlineDevices int       `json:"online_devices"`
}

// 流量统计响应
type TrafficStatsResponse struct {
	TimeRange string         `json:"time_range"`
	Data      []*TrafficPoint `json:"data"`
	Total     int            `json:"total"`
}

type TrafficPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Upload   int64     `json:"upload"`
	Download int64     `json:"download"`
}

// 设备分布响应
type DeviceDistributionResponse struct {
	Distribution []*PlatformDistribution `json:"distribution"`
	Total        int                    `json:"total"`
}

type PlatformDistribution struct {
	Platform   string  `json:"platform"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// 告警趋势响应
type AlertTrendResponse struct {
	TimeRange string             `json:"time_range"`
	Data      []*AlertTrendPoint `json:"data"`
	Total     int                `json:"total"`
}

type AlertTrendPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Critical  int       `json:"critical"`
	High      int       `json:"high"`
	Medium    int       `json:"medium"`
	Low       int       `json:"low"`
}

// 拓扑设备响应
type TopologyDevicesResponse struct {
	Devices []*TopologyDevice `json:"devices"`
	Total   int                `json:"total"`
}

type TopologyDevice struct {
	ID                uuid.UUID  `json:"id"`
	Name              string     `json:"name"`
	VirtualIP         string     `json:"virtual_ip"`
	Platform          string     `json:"platform"`
	NATType           string     `json:"nat_type"`
	IsOnline          bool       `json:"is_online"`
	LastSeen          *time.Time `json:"last_seen"`
	VirtualNetworkID  uuid.UUID  `json:"virtual_network_id"`
}

// 拓扑对等响应
type TopologyPeersResponse struct {
	Peers []*TopologyPeer `json:"peers"`
	Total int              `json:"total"`
}

type TopologyPeer struct {
	ID             uuid.UUID  `json:"id"`
	DeviceAID      uuid.UUID  `json:"device_a_id"`
	DeviceBID      uuid.UUID  `json:"device_b_id"`
	ConnectionType string     `json:"connection_type"`
	Latency        *int       `json:"latency"`
	LastHandshake  *time.Time `json:"last_handshake"`
	StartedAt      time.Time  `json:"started_at"`
	EndpointA      *string    `json:"endpoint_a"`
	EndpointB      *string    `json:"endpoint_b"`
}
