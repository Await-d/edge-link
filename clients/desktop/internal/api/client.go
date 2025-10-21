package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client API客户端
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient 创建API客户端
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RegisterDeviceRequest 设备注册请求
type RegisterDeviceRequest struct {
	PreSharedKey string `json:"pre_shared_key"`
	Name         string `json:"name"`
	Platform     string `json:"platform"`
	PublicKey    string `json:"public_key"`
}

// RegisterDeviceResponse 设备注册响应
type RegisterDeviceResponse struct {
	DeviceID      string   `json:"device_id"`
	VirtualIP     string   `json:"virtual_ip"`
	VirtualSubnet string   `json:"virtual_subnet"`
	Peers         []Peer   `json:"peers"`
}

// Peer 对等设备
type Peer struct {
	PublicKey  string `json:"public_key"`
	VirtualIP  string `json:"virtual_ip"`
	Endpoint   string `json:"endpoint,omitempty"`
}

// DeviceConfigResponse 设备配置响应
type DeviceConfigResponse struct {
	VirtualIP     string `json:"virtual_ip"`
	VirtualSubnet string `json:"virtual_subnet"`
	Peers         []Peer `json:"peers"`
}

// MetricsRequest 指标提交请求
type MetricsRequest struct {
	BytesSent     int64 `json:"bytes_sent"`
	BytesReceived int64 `json:"bytes_received"`
	LatencyMs     int   `json:"latency_ms,omitempty"`
}

// RegisterDevice 注册设备
func (c *Client) RegisterDevice(req *RegisterDeviceRequest) (*RegisterDeviceResponse, error) {
	url := fmt.Sprintf("%s/api/v1/device/register", c.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("registration failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var response RegisterDeviceResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// GetDeviceConfig 获取设备配置
func (c *Client) GetDeviceConfig(deviceID string) (*DeviceConfigResponse, error) {
	url := fmt.Sprintf("%s/api/v1/device/%s/config", c.baseURL, deviceID)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get config failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var response DeviceConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// SubmitMetrics 提交设备指标
func (c *Client) SubmitMetrics(deviceID string, metrics *MetricsRequest) error {
	url := fmt.Sprintf("%s/api/v1/device/%s/metrics", c.baseURL, deviceID)

	body, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to submit metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("submit metrics failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}
