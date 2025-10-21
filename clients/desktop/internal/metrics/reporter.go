package metrics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// DeviceMetrics 设备指标
type DeviceMetrics struct {
	DeviceID       string             `json:"device_id"`
	Online         bool               `json:"online"`
	BytesSent      int64              `json:"bytes_sent"`
	BytesReceived  int64              `json:"bytes_received"`
	LatencyMs      map[string]int     `json:"latency_ms"`      // peerID -> latency
	PacketLoss     map[string]float64 `json:"packet_loss"`     // peerID -> loss rate
	PublicEndpoint string             `json:"public_endpoint,omitempty"`
	Timestamp      time.Time          `json:"timestamp"`
}

// Reporter 指标报告器
type Reporter struct {
	deviceID        string
	controlPlaneURL string
	httpClient      *http.Client
	interval        time.Duration
	stopCh          chan struct{}
}

// NewReporter 创建指标报告器
func NewReporter(deviceID, controlPlaneURL string, interval time.Duration) *Reporter {
	return &Reporter{
		deviceID:        deviceID,
		controlPlaneURL: controlPlaneURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start 启动定期指标上报
func (r *Reporter) Start() {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := r.reportMetrics(); err != nil {
				// 记录错误但不停止
				fmt.Printf("Failed to report metrics: %v\n", err)
			}
		case <-r.stopCh:
			return
		}
	}
}

// Stop 停止指标上报
func (r *Reporter) Stop() {
	close(r.stopCh)
}

// reportMetrics 上报指标
func (r *Reporter) reportMetrics() error {
	// 收集指标
	metrics, err := r.collectMetrics()
	if err != nil {
		return fmt.Errorf("failed to collect metrics: %w", err)
	}

	// 发送到控制平面
	if err := r.sendMetrics(metrics); err != nil {
		return fmt.Errorf("failed to send metrics: %w", err)
	}

	return nil
}

// collectMetrics 收集设备指标
func (r *Reporter) collectMetrics() (*DeviceMetrics, error) {
	// TODO: 实现完整的指标收集逻辑
	// 当前为简化实现，生产环境需要：
	// 1. 从WireGuard接口读取流量统计（wg show dump）
	// 2. 测量到各对等设备的延迟（ICMP ping或UDP探测）
	// 3. 计算丢包率（基于WireGuard的handshake超时）
	// 4. 获取本地公网端点（通过STUN）

	metrics := &DeviceMetrics{
		DeviceID:      r.deviceID,
		Online:        true,
		BytesSent:     0,     // 占位符
		BytesReceived: 0,     // 占位符
		LatencyMs:     make(map[string]int),
		PacketLoss:    make(map[string]float64),
		Timestamp:     time.Now(),
	}

	return metrics, nil
}

// sendMetrics 发送指标到控制平面
func (r *Reporter) sendMetrics(metrics *DeviceMetrics) error {
	// 构建API端点
	url := fmt.Sprintf("%s/api/v1/device/%s/metrics", r.controlPlaneURL, r.deviceID)

	// 序列化指标
	data, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	// 发送POST请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	// TODO: 添加设备签名认证
	// req.Header.Set("Authorization", "Bearer "+deviceSignature)

	// 发送请求
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// ReportHeartbeat 上报心跳
func (r *Reporter) ReportHeartbeat() error {
	metrics := &DeviceMetrics{
		DeviceID:  r.deviceID,
		Online:    true,
		Timestamp: time.Now(),
	}

	return r.sendMetrics(metrics)
}

// ReportConnectionMetrics 上报连接指标
func (r *Reporter) ReportConnectionMetrics(peerID string, latencyMs int, packetLoss float64) error {
	metrics := &DeviceMetrics{
		DeviceID:   r.deviceID,
		Online:     true,
		LatencyMs:  map[string]int{peerID: latencyMs},
		PacketLoss: map[string]float64{peerID: packetLoss},
		Timestamp:  time.Now(),
	}

	return r.sendMetrics(metrics)
}

// ReportTrafficStats 上报流量统计
func (r *Reporter) ReportTrafficStats(bytesSent, bytesReceived int64) error {
	metrics := &DeviceMetrics{
		DeviceID:      r.deviceID,
		Online:        true,
		BytesSent:     bytesSent,
		BytesReceived: bytesReceived,
		Timestamp:     time.Now(),
	}

	return r.sendMetrics(metrics)
}
