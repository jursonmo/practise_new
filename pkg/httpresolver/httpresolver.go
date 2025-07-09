package httpresolver

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jursonmo/practise_new/pkg/combinederror"
)

var DefaultFallbackResolver *FallbackResolver
var defaultHttpClient *http.Client

func init() {
	DefaultFallbackResolver = NewFallbackResolver()
	defaultHttpClient = &http.Client{
		Transport: NewHttpResolverTransport(DefaultFallbackResolver, nil),
		Timeout:   10 * time.Second,
	}
}

func AddDomainIp(domain string, ips ...string) {
	DefaultFallbackResolver.AddPreset(domain, ips...)
}

func LoadDomains(filePath string) error {
	return DefaultFallbackResolver.LoadDomains(filePath)
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

func (r *FallbackResolver) LoadDomains(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 使用scanner逐行读取文件
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// 分割域名和IP部分
		parts := strings.Fields(line)
		if len(parts) < 2 {
			fmt.Printf("无效的行格式: %s\n", line)
			continue
		}

		domain := parts[0]
		ipString := strings.Join(parts[1:], " ") // 合并后面的部分(处理可能有多个空格的情况)

		// 去除IP字符串中的逗号，然后分割成IP列表
		ipString = strings.ReplaceAll(ipString, ",", " ")
		ips := strings.Fields(ipString)

		// 打印结果(在实际应用中，你可以在这里使用domain和ips变量)
		fmt.Printf("域名: %s\n", domain)
		fmt.Printf("IP地址: %v\n", ips)
		fmt.Println("------")
		r.AddPreset(domain, ips...)
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("读取文件出错:", err)
	}
	return nil
}

// 自定义解析逻辑
func (r *FallbackResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	// 1. 先尝试系统 DNS 解析
	// 域名解释的超时时间 不要超过client timeout 设定的超时时间，不然域名解释失败后，留给后续使用指定ip 连接的时间就不够了
	var dnsTimeout time.Duration
	deadline, ok := ctx.Deadline()
	if ok {
		// ctx 设置了截止时间
		timeRemaining := time.Until(deadline) // 计算剩余时间, 默认client.Timeout是10秒的话,这里就是10秒
		//fmt.Printf("Context will expire at: %v, remaining: %v\n", deadline, timeRemaining)
		// if timeRemaining >= 5*time.Second {
		// 	dnsTimeout = timeRemaining - 2*time.Second //至少留两秒来连接服务器
		// } else if timeRemaining >= 4*time.Second {
		// 	dnsTimeout = timeRemaining - 1500*time.Millisecond //留1.5秒连接服务器
		// }
		dnsTimeout = timeRemaining * 2 / 3 //三分之二的时间用于dns查询，最大不能超过5秒，超过5秒就用自己预设的ip来连接了
	}

	//如果不设置超时或者超时时间超过5秒，统一设置成5秒
	if dnsTimeout == 0 || dnsTimeout > 5*time.Second {
		dnsTimeout = 5 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, dnsTimeout)
	defer cancel()
	ips, err := r.systemResolver.LookupHost(ctx, host)
	if err == nil {
		return ips, nil
	}
	fmt.Printf("take %v system dns get host:%s fail, try to get ip from presetIPs", dnsTimeout, host)
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
			combinederror := combinederror.NewCombinedError()
			for i, ip := range ips {
				conn, err := net.DialTimeout(network, net.JoinHostPort(ip, port), 2*time.Second)
				if err == nil {
					if i != 0 {
						//如果前面的ip是连不上的，那么现在这个ip 连上了，放在第一位，以后优先选它
						//switch with ip0
						ips[i], ips[0] = ips[0], ips[i]
					}
					return conn, nil
				}
				combinederror.Append(err)
			}
			return nil, combinederror
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

func GetDefaultHttpClient() *http.Client {
	return defaultHttpClient
}

func GetDefaultTransport() http.RoundTripper {
	return defaultHttpClient.Transport
}
