package client

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/yinxulai/ait/internal/server/types"
)

func newMeasuredTransport(config types.Input) *http.Transport {
	transport := &http.Transport{
		DisableKeepAlives:  true,
		DisableCompression: false,
		Proxy:              http.ProxyFromEnvironment,
	}

	proxyURL := strings.TrimSpace(config.ProxyURL)
	if proxyURL == "" {
		return transport
	}

	parsed, err := url.Parse(proxyURL)
	if err == nil && parsed.Scheme != "" && parsed.Host != "" {
		transport.Proxy = http.ProxyURL(parsed)
		return transport
	}

	transport.Proxy = func(*http.Request) (*url.URL, error) {
		return nil, fmt.Errorf("invalid proxy_url: %s", proxyURL)
	}
	return transport
}
