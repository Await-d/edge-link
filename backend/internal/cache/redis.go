package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/edgelink/backend/internal/config"
	"github.com/redis/go-redis/v9"
)

// RedisClient Redis客户端包装
type RedisClient struct {
	client *redis.Client
}

// New 创建Redis客户端
func New(cfg *config.RedisConfig) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisClient{client: client}, nil
}

// Get 获取值
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

// Set 设置值（带过期时间）
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

// Delete 删除键
func (r *RedisClient) Delete(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

// Exists 检查键是否存在
func (r *RedisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Exists(ctx, keys...).Result()
}

// Publish 发布消息
func (r *RedisClient) Publish(ctx context.Context, channel string, message interface{}) error {
	return r.client.Publish(ctx, channel, message).Err()
}

// Subscribe 订阅频道
func (r *RedisClient) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return r.client.Subscribe(ctx, channels...)
}

// Close 关闭连接
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// Client 获取原始Redis客户端（用于高级操作）
func (r *RedisClient) Client() *redis.Client {
	return r.client
}

// ======================== 缓存策略 ========================

// 缓存键前缀
const (
	// 设备在线状态缓存（TTL: 5分钟）
	KeyDeviceOnlineStatus = "device:online:"
	// 对等配置缓存（TTL: 10分钟）
	KeyPeerConfig = "peer:config:"
	// 设备配置缓存（TTL: 10分钟）
	KeyDeviceConfig = "device:config:"
	// 虚拟网络配置缓存（TTL: 30分钟）
	KeyVirtualNetworkConfig = "vnet:config:"
	// 速率限制计数器（TTL: 1分钟）
	KeyRateLimit = "ratelimit:"
	// Session令牌缓存（TTL: 与session过期时间一致）
	KeySessionToken = "session:token:"
	// NAT类型检测结果缓存（TTL: 1小时）
	KeyNATDetection = "nat:detection:"
)

// 缓存TTL策略
const (
	// 短期缓存（频繁变化的数据）
	TTLShort = 5 * time.Minute
	// 中期缓存（定期更新的数据）
	TTLMedium = 10 * time.Minute
	// 长期缓存（相对静态的数据）
	TTLLong = 30 * time.Minute
	// 超长期缓存（很少变化的数据）
	TTLVeryLong = 1 * time.Hour
	// 速率限制窗口
	TTLRateLimit = 1 * time.Minute
)

var (
	// ErrCacheMiss 缓存未命中
	ErrCacheMiss = errors.New("cache miss")
	// ErrInvalidCacheData 缓存数据无效
	ErrInvalidCacheData = errors.New("invalid cache data")
)

// SetJSON 设置JSON对象（自动序列化）
func (r *RedisClient) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal json: %w", err)
	}
	return r.Set(ctx, key, data, expiration)
}

// GetJSON 获取JSON对象（自动反序列化）
func (r *RedisClient) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := r.Get(ctx, key)
	if err != nil {
		if err == redis.Nil {
			return ErrCacheMiss
		}
		return err
	}

	if err := json.Unmarshal([]byte(data), dest); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidCacheData, err)
	}

	return nil
}

// SetDeviceOnlineStatus 设置设备在线状态
func (r *RedisClient) SetDeviceOnlineStatus(ctx context.Context, deviceID string, isOnline bool) error {
	key := KeyDeviceOnlineStatus + deviceID
	var value string
	if isOnline {
		value = "1"
	} else {
		value = "0"
	}
	return r.Set(ctx, key, value, TTLShort)
}

// GetDeviceOnlineStatus 获取设备在线状态
func (r *RedisClient) GetDeviceOnlineStatus(ctx context.Context, deviceID string) (bool, error) {
	key := KeyDeviceOnlineStatus + deviceID
	result, err := r.Get(ctx, key)
	if err != nil {
		if err == redis.Nil {
			return false, ErrCacheMiss
		}
		return false, err
	}
	return result == "1", nil
}

// SetDeviceConfig 缓存设备配置
func (r *RedisClient) SetDeviceConfig(ctx context.Context, deviceID string, config interface{}) error {
	key := KeyDeviceConfig + deviceID
	return r.SetJSON(ctx, key, config, TTLMedium)
}

// GetDeviceConfig 获取设备配置
func (r *RedisClient) GetDeviceConfig(ctx context.Context, deviceID string, dest interface{}) error {
	key := KeyDeviceConfig + deviceID
	return r.GetJSON(ctx, key, dest)
}

// InvalidateDeviceConfig 使设备配置缓存失效
func (r *RedisClient) InvalidateDeviceConfig(ctx context.Context, deviceID string) error {
	key := KeyDeviceConfig + deviceID
	return r.Delete(ctx, key)
}

// SetPeerConfig 缓存对等配置
func (r *RedisClient) SetPeerConfig(ctx context.Context, deviceID string, peerConfig interface{}) error {
	key := KeyPeerConfig + deviceID
	return r.SetJSON(ctx, key, peerConfig, TTLMedium)
}

// GetPeerConfig 获取对等配置
func (r *RedisClient) GetPeerConfig(ctx context.Context, deviceID string, dest interface{}) error {
	key := KeyPeerConfig + deviceID
	return r.GetJSON(ctx, key, dest)
}

// InvalidatePeerConfig 使对等配置缓存失效
func (r *RedisClient) InvalidatePeerConfig(ctx context.Context, deviceID string) error {
	key := KeyPeerConfig + deviceID
	return r.Delete(ctx, key)
}

// SetVirtualNetworkConfig 缓存虚拟网络配置
func (r *RedisClient) SetVirtualNetworkConfig(ctx context.Context, networkID string, config interface{}) error {
	key := KeyVirtualNetworkConfig + networkID
	return r.SetJSON(ctx, key, config, TTLLong)
}

// GetVirtualNetworkConfig 获取虚拟网络配置
func (r *RedisClient) GetVirtualNetworkConfig(ctx context.Context, networkID string, dest interface{}) error {
	key := KeyVirtualNetworkConfig + networkID
	return r.GetJSON(ctx, key, dest)
}

// InvalidateVirtualNetworkConfig 使虚拟网络配置缓存失效
func (r *RedisClient) InvalidateVirtualNetworkConfig(ctx context.Context, networkID string) error {
	key := KeyVirtualNetworkConfig + networkID
	return r.Delete(ctx, key)
}

// IncrementRateLimit 增加速率限制计数器
// 返回当前计数，如果超过限制则返回错误
func (r *RedisClient) IncrementRateLimit(ctx context.Context, identifier string, limit int64) (int64, error) {
	key := KeyRateLimit + identifier

	// 使用Redis管道执行原子操作
	pipe := r.client.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, TTLRateLimit)

	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}

	count := incr.Val()
	if count > limit {
		return count, fmt.Errorf("rate limit exceeded: %d/%d", count, limit)
	}

	return count, nil
}

// GetRateLimitCount 获取当前速率限制计数
func (r *RedisClient) GetRateLimitCount(ctx context.Context, identifier string) (int64, error) {
	key := KeyRateLimit + identifier
	result, err := r.client.Get(ctx, key).Int64()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, err
	}
	return result, nil
}

// SetNATDetectionResult 缓存NAT类型检测结果
func (r *RedisClient) SetNATDetectionResult(ctx context.Context, deviceID string, natType interface{}) error {
	key := KeyNATDetection + deviceID
	return r.SetJSON(ctx, key, natType, TTLVeryLong)
}

// GetNATDetectionResult 获取NAT类型检测结果
func (r *RedisClient) GetNATDetectionResult(ctx context.Context, deviceID string, dest interface{}) error {
	key := KeyNATDetection + deviceID
	return r.GetJSON(ctx, key, dest)
}

// ======================== 缓存预热和失效策略 ========================

// WarmupCache 预热常用缓存
// 在系统启动时调用，提前加载热点数据
func (r *RedisClient) WarmupCache(ctx context.Context, warmupFunc func(context.Context, *RedisClient) error) error {
	log.Println("Starting cache warmup...")
	start := time.Now()

	if err := warmupFunc(ctx, r); err != nil {
		return fmt.Errorf("cache warmup failed: %w", err)
	}

	log.Printf("Cache warmup completed in %v", time.Since(start))
	return nil
}

// InvalidatePattern 根据模式批量失效缓存
// 例如：InvalidatePattern(ctx, "device:*") 失效所有设备相关缓存
func (r *RedisClient) InvalidatePattern(ctx context.Context, pattern string) error {
	iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()
	keys := make([]string, 0)

	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return err
	}

	if len(keys) > 0 {
		log.Printf("Invalidating %d keys matching pattern: %s", len(keys), pattern)
		return r.Delete(ctx, keys...)
	}

	return nil
}

// GetCacheStats 获取缓存统计信息
func (r *RedisClient) GetCacheStats(ctx context.Context) (map[string]interface{}, error) {
	info, err := r.client.Info(ctx, "stats").Result()
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"info": info,
	}

	// 获取键空间信息
	dbInfo, err := r.client.Info(ctx, "keyspace").Result()
	if err == nil {
		stats["keyspace"] = dbInfo
	}

	return stats, nil
}
