package httpresolver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

var DefaultFallbackResolver *FallbackResolver

func init() {
	DefaultFallbackResolver = NewFallbackResolver()
}

func AddDomainIp(domain string, ips ...string) {
	DefaultFallbackResolver.AddPreset(domain, ips...)
}

// 自定义 DNS 解析器
type FallbackResolver struct {
	systemResolver *net.Resolver
	presetIPs      map[string][]string // 域名 -> 预设 IP
	mu             sync.RWMutex
}

func NewFallbackResolver() *FallbackResolver {
	return &FallbackResolver{
		systemResolver: &net.Resolver{
			PreferGo: false, // 优先使用系统 DNS
		},
		presetIPs: make(map[string][]string),
	}
}

// 添加预设 IP 映射
func (r *FallbackResolver) AddPreset(domain string, ips ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.presetIPs[domain] = ips
}

// 自定义解析逻辑
func (r *FallbackResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	// 1. 先尝试系统 DNS 解析
	ips, err := r.systemResolver.LookupHost(ctx, host)
	if err == nil {
		return ips, nil
	}

	// 2. 系统解析失败时检查预设 IP
	r.mu.RLock()
	presetIP, ok := r.presetIPs[host]
	r.mu.RUnlock()

	if ok {
		return presetIP, nil
	}

	//3. 如果精准无法匹配，采用后缀匹配, 即请求 abc.example.com 可以返回 example.com 预设的ip
	for domain, ips := range r.presetIPs {
		if strings.HasSuffix(host, domain) {
			return ips, nil
		}
	}

	// 4. 返回错误
	return nil, fmt.Errorf("no preset IP for %s", host)
}

func NewHttpResolverTransport(resolver *FallbackResolver, printResolveResult func(host string, ip []string)) *http.Transport {
	if resolver == nil {
		resolver = DefaultFallbackResolver
	}
	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}

			// 使用自定义解析器解析域名
			ips, err := resolver.LookupHost(ctx, host)
			if err != nil {
				return nil, err
			}

			if printResolveResult != nil {
				printResolveResult(host, ips)
			}

			// 尝试所有解析到的 IP
			var lastErr error
			for _, ip := range ips {
				conn, err := net.DialTimeout(network, net.JoinHostPort(ip, port), 2*time.Second)
				if err == nil {
					return conn, nil
				}
				lastErr = err
			}
			return nil, lastErr
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

// 创建自定义 HTTP 客户端
func NewHttpResolverClient(resolver *FallbackResolver, printResolveResult func(host string, ip []string)) *http.Client {
	return &http.Client{
		Transport: NewHttpResolverTransport(resolver, printResolveResult),
		Timeout:   10 * time.Second,
	}
}
