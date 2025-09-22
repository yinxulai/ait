package network

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// GetPublicIP 获取公网出口 IP 地址
// 依次尝试多个服务，确保获取成功
func GetPublicIP() (string, error) {
	// IP 查询服务列表（按可靠性排序）
	services := []struct {
		name string
		url  string
	}{
		{"ipify", "https://api.ipify.org"},
		{"icanhazip", "https://icanhazip.com"},
		{"ifconfig.me", "https://ifconfig.me/ip"},
		{"ident.me", "https://ident.me"},
		{"checkip", "https://checkip.amazonaws.com"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, service := range services {
		ip, err := queryIPService(ctx, service.url)
		if err == nil && ip != "" {
			return ip, nil
		}
		// 如果失败，继续尝试下一个服务
	}

	return "", fmt.Errorf("failed to get public IP from all services")
}

// queryIPService 查询单个 IP 服务
func queryIPService(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	// 设置用户代理
	req.Header.Set("User-Agent", "ait-tool/1.0")

	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	ip := strings.TrimSpace(string(body))
	
	// 简单验证 IP 格式（检查是否包含数字和点）
	if len(ip) < 7 || len(ip) > 15 || !strings.Contains(ip, ".") {
		return "", fmt.Errorf("invalid IP format: %s", ip)
	}

	return ip, nil
}

// GetPublicIPCached 获取公网 IP（带缓存）
var cachedIP string
var lastFetchTime time.Time
var cacheDuration = 5 * time.Minute

func GetPublicIPCached() (string, error) {
	now := time.Now()
	
	// 如果缓存有效，直接返回
	if cachedIP != "" && now.Sub(lastFetchTime) < cacheDuration {
		return cachedIP, nil
	}
	
	// 获取新的 IP
	ip, err := GetPublicIP()
	if err != nil {
		// 如果获取失败但有缓存，返回缓存值
		if cachedIP != "" {
			return cachedIP, nil
		}
		return "", err
	}
	
	// 更新缓存
	cachedIP = ip
	lastFetchTime = now
	
	return ip, nil
}
