package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/edgelink/backend/internal/service"
	"github.com/edgelink/backend/internal/auth"
	"github.com/edgelink/backend/internal/crypto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DeviceHandler 设备相关HTTP处理器
type DeviceHandler struct {
	deviceService   *service.DeviceService
	topologyService *service.TopologyService
	pskAuth         *auth.PSKAuthenticator
}

// NewDeviceHandler 创建设备处理器实例
func NewDeviceHandler(
	deviceService *service.DeviceService,
	topologyService *service.TopologyService,
	pskAuth *auth.PSKAuthenticator,
) *DeviceHandler {
	return &DeviceHandler{
		deviceService:   deviceService,
		topologyService: topologyService,
		pskAuth:         pskAuth,
	}
}

// RegisterDevice godoc
// @Summary      注册新设备
// @Description  使用预共享密钥注册新设备到虚拟网络
// @Tags         devices
// @Accept       json
// @Produce      json
// @Param        X-Pre-Shared-Key  header  string  true  "预共享密钥"
// @Param        request           body    service.RegisterDeviceRequest  true  "注册请求"
// @Success      201  {object}  service.RegisterDeviceResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/device/register [post]
func (h *DeviceHandler) RegisterDevice(c *gin.Context) {
	// 1. 提取预共享密钥
	psk := c.GetHeader("X-Pre-Shared-Key")
	if psk == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "missing_psk",
			Message: "X-Pre-Shared-Key header is required",
		})
		return
	}

	// 2. 解析请求体
	var req service.RegisterDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: fmt.Sprintf("failed to parse request: %v", err),
		})
		return
	}

	// 3. 将PSK添加到请求中
	req.PreSharedKey = psk

	// 4. 调用服务层注册设备
	resp, err := h.deviceService.RegisterDevice(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "registration_failed",
			Message: err.Error(),
		})
		return
	}

	// 5. 返回成功响应
	c.JSON(http.StatusCreated, resp)
}

// GetDeviceConfig godoc
// @Summary      获取设备配置
// @Description  获取设备的WireGuard配置（包含对等设备列表）
// @Tags         devices
// @Accept       json
// @Produce      json
// @Param        device_id  path  string  true  "设备ID"
// @Param        Authorization  header  string  true  "Bearer {device_signature}"
// @Success      200  {object}  DeviceConfigResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /api/v1/device/{device_id}/config [get]
func (h *DeviceHandler) GetDeviceConfig(c *gin.Context) {
	// 1. 解析设备ID
	deviceIDStr := c.Param("device_id")
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_device_id",
			Message: "device_id must be a valid UUID",
		})
		return
	}

	// 2. 验证设备身份（通过签名）
	// TODO: 实现完整的设备签名验证
	// 当前为简化实现，生产环境需要：
	// 1. 从Authorization header提取签名
	// 2. 验证签名是否由设备私钥生成
	// 3. 检查签名时间戳防止重放攻击

	// 3. 获取设备信息
	device, err := h.deviceService.GetDeviceConfig(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "device_not_found",
			Message: err.Error(),
		})
		return
	}

	// 4. 获取对等配置
	peers, err := h.topologyService.GetPeerConfigurations(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "failed_to_get_peers",
			Message: err.Error(),
		})
		return
	}

	// 5. 构建响应
	resp := DeviceConfigResponse{
		DeviceID:         device.ID,
		VirtualIP:        device.VirtualIP,
		VirtualNetworkID: device.VirtualNetworkID,
		Platform:         string(device.Platform),
		Peers:            peers,
		UpdatedAt:        device.UpdatedAt,
	}

	c.JSON(http.StatusOK, resp)
}

// SubmitDeviceMetrics godoc
// @Summary      提交设备指标
// @Description  设备上报运行状态和网络指标
// @Tags         devices
// @Accept       json
// @Produce      json
// @Param        device_id  path  string  true  "设备ID"
// @Param        Authorization  header  string  true  "Bearer {device_signature}"
// @Param        metrics  body  DeviceMetricsRequest  true  "设备指标"
// @Success      200  {object}  SuccessResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Router       /api/v1/device/{device_id}/metrics [post]
func (h *DeviceHandler) SubmitDeviceMetrics(c *gin.Context) {
	// 1. 解析设备ID
	deviceIDStr := c.Param("device_id")
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_device_id",
			Message: "device_id must be a valid UUID",
		})
		return
	}

	// 2. 验证设备身份
	// TODO: 实现设备签名验证（同GetDeviceConfig）

	// 3. 解析指标数据
	var metrics DeviceMetricsRequest
	if err := c.ShouldBindJSON(&metrics); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_metrics",
			Message: fmt.Sprintf("failed to parse metrics: %v", err),
		})
		return
	}

	// 4. 更新设备在线状态
	if err := h.deviceService.UpdateDeviceStatus(c.Request.Context(), deviceID, metrics.Online); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "failed_to_update_status",
			Message: err.Error(),
		})
		return
	}

	// 5. 存储指标数据
	// TODO: 实现完整的指标存储逻辑
	// 当前为简化实现，实际生产需要：
	// 1. 将指标数据写入时序数据库（如Prometheus/InfluxDB）
	// 2. 触发监控告警规则
	// 3. 更新设备最后活跃时间

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "metrics submitted successfully",
	})
}

// DeviceConfigResponse 设备配置响应
type DeviceConfigResponse struct {
	DeviceID         uuid.UUID                      `json:"device_id"`
	VirtualIP        string                         `json:"virtual_ip"`
	VirtualNetworkID uuid.UUID                      `json:"virtual_network_id"`
	Platform         string                         `json:"platform"`
	Peers            []crypto.WireGuardPeerConfig  `json:"peers"`
	UpdatedAt        time.Time                      `json:"updated_at"`
}

// DeviceMetricsRequest 设备指标请求
type DeviceMetricsRequest struct {
	Online          bool              `json:"online"`
	BytesSent       int64             `json:"bytes_sent"`
	BytesReceived   int64             `json:"bytes_received"`
	LatencyMs       map[string]int    `json:"latency_ms"` // peerID -> latency
	PacketLoss      map[string]float64 `json:"packet_loss"` // peerID -> loss rate
	PublicEndpoint  string            `json:"public_endpoint,omitempty"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// SuccessResponse 成功响应
type SuccessResponse struct {
	Message string `json:"message"`
}
