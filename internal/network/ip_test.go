package network

import (
	"testing"
	"time"
)

func TestGetPublicIP(t *testing.T) {
	ip, err := GetPublicIP()
	if err != nil {
		t.Fatalf("GetPublicIP() failed: %v", err)
	}
	
	if ip == "" {
		t.Fatal("GetPublicIP() returned empty IP")
	}
	
	// 简单验证 IP 格式
	if len(ip) < 7 || len(ip) > 15 {
		t.Errorf("Invalid IP length: %s", ip)
	}
	
	t.Logf("Public IP: %s", ip)
}

func TestGetPublicIPCached(t *testing.T) {
	// 第一次调用
	start := time.Now()
	ip1, err := GetPublicIPCached()
	duration1 := time.Since(start)
	if err != nil {
		t.Fatalf("First GetPublicIPCached() failed: %v", err)
	}
	
	// 第二次调用（应该使用缓存）
	start = time.Now()
	ip2, err := GetPublicIPCached()
	duration2 := time.Since(start)
	if err != nil {
		t.Fatalf("Second GetPublicIPCached() failed: %v", err)
	}
	
	// 验证结果一致
	if ip1 != ip2 {
		t.Errorf("Cached IP mismatch: %s != %s", ip1, ip2)
	}
	
	// 验证第二次调用更快（使用了缓存）
	if duration2 >= duration1 {
		t.Logf("Warning: Second call took %v, first call took %v (expected second to be faster)", duration2, duration1)
	}
	
	t.Logf("First call: %v, Second call: %v, IP: %s", duration1, duration2, ip1)
}
