package shared

import (
	"fmt"
	"strings"
	"time"

	"github.com/yinxulai/ait/internal/i18n"
)

// FmtDuration 格式化持续时间为友好的文本（如"1h 23m 45s"）。
func FmtDuration(d time.Duration) string {
	if d < time.Second {
		return "< 1s"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %02dm %02ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// FmtRelativeTime 返回相对时间的友好文本（如"2小时前"、"刚刚"）。
func FmtRelativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	if d < 0 {
		d = 0
	}

	switch {
	case d < time.Minute:
		return i18n.T(i18n.KJustNow)
	case d < time.Hour:
		return fmt.Sprintf(i18n.T(i18n.KMinutesAgoFmt), int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf(i18n.T(i18n.KHoursAgoFmt), int(d.Hours()))
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		return fmt.Sprintf(i18n.T(i18n.KDaysAgoFmt), days)
	default:
		return t.Format("2006-01-02")
	}
}

// RunStatusText 返回运行状态的国际化文本。
func RunStatusText(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "running":
		return i18n.T(i18n.KRunning)
	case "completed":
		return i18n.T(i18n.KCompleted)
	case "stopped":
		return i18n.T(i18n.KStopped)
	case "failed":
		return i18n.T(i18n.KRunFailed)
	case "":
		return i18n.T(i18n.KWaitingStatus)
	default:
		return status
	}
}

// ModeShortLabel 返回模式的简短标签。
func ModeShortLabel(mode string) string {
	switch strings.ToLower(mode) {
	case "turbo":
		return "T"
	case "integrity":
		return "I"
	default:
		return "S"
	}
}
