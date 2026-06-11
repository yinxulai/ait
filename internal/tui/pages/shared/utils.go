package shared

import "github.com/yinxulai/ait/internal/server"

var AppVersion string

// SetAppVersion 设置应用版本号。
func SetAppVersion(v string) {
	AppVersion = v
}

// IsRunStateRunning 判断运行状态是否为运行中。
func IsRunStateRunning(rs *server.RunState) bool {
	if rs == nil {
		return false
	}
	return rs.Status == server.RunStatusRunning
}

// MinInt 返回两个整数中较小的一个。
func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MaxInt 返回两个整数中较大的一个。
func MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
