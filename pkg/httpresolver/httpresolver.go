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
	mu             sync.RWMutex
	resolveCache   map[string][]string
	presetIPs      map[string][]string // 域名 -> 预设 IP
}

func NewFallbackResolver() *FallbackResolver {
	return &FallbackResolver{
		systemResolver: &net.Resolver{
			PreferGo: false, // 优先使用系统 DNS
		},
		presetIPs:    make(map[string][]string),
		resolveCache: make(map[string][]string),
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

		fmt.Printf("加载域名: %s, ips:%v\n", domain, ips)
		fmt.Println("------")
		r.AddPreset(domain, ips...)
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("读取文件出错:", err)
	}
	return nil
}

// 自定义解析逻辑
func (r *FallbackResolver) LookupHost(ctx context.Context, host string) (ips []string, inPreSet bool, inCache bool, err error) {
	// 1. 先尝试系统 DNS 解析
	// 域名解释的超时时间 不要超过client timeout 设定的超时时间，不然域名解释失败后，留给后续使用指定ip 连接的时间就不够了
	var dnsTimeout time.Duration
	deadline, ok := ctx.Deadline()
	if ok {
		// ctx 设置了截止时间
		timeRemaining := time.Until(deadline) // 计算剩余时间, 如果设置client.Timeout是10秒的话,这里就是10秒
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
	ips, err = r.systemResolver.LookupHost(ctx, host)
	if err == nil {
		r.mu.RLock()
		r.resolveCache[host] = ips
		r.mu.RUnlock()
		return ips, false, false, nil
	}
	fmt.Printf(" system dns take %v to get host:%s fail, err:%v, try to get ip from presetIPs or cache\n", dnsTimeout, host, err)

	// 2. 系统解析失败时检查预设 IP
	r.mu.RLock()
	defer r.mu.RUnlock()
	presetIP, ok := r.presetIPs[host]

	if ok {
		return presetIP, true, false, nil
	}

	//3. 如果精准无法匹配，采用后缀匹配, 即请求 abc.example.com 可以返回 example.com 预设的ip
	for domain, ips := range r.presetIPs {
		if strings.HasSuffix(host, domain) {
			return ips, true, false, nil
		}
	}

	//4. 尝试返回之前成功的ip
	if ips, ok := r.resolveCache[host]; ok {
		return ips, false, true, nil
	}

	// 5. 返回错误
	return nil, false, false, fmt.Errorf("no preset IP for %s", host)
}

func NewHttpResolverTransport(resolver *FallbackResolver, printResolveResult func(host string, ip []string, inPreset, inCache bool)) *http.Transport {
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
			ips, inPreSet, inCache, err := resolver.LookupHost(ctx, host)
			if err != nil {
				return nil, err
			}

			if printResolveResult != nil {
				printResolveResult(host, ips, inPreSet, inCache)
			}

			// 尝试所有解析到的 IP
			combinederror := combinederror.NewCombinedError()
			for i, ip := range ips {
				conn, err := net.DialTimeout(network, net.JoinHostPort(ip, port), 2*time.Second)
				if err == nil {
					if i != 0 {
						resolver.mu.Lock()
						//如果前面的ip是连不上的，那么现在这个ip 连上了，放在第一位，以后优先选它
						//switch with ip0
						ips[i], ips[0] = ips[0], ips[i]
						if inPreSet {
							resolver.presetIPs[host] = ips
						}
						if inCache {
							resolver.resolveCache[host] = ips
						}
						resolver.mu.Unlock()
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
func NewHttpResolverClient(resolver *FallbackResolver, printResolveResult func(host string, ip []string, inPreset, inCache bool)) *http.Client {
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
