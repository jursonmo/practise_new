package router

import (
	"fmt"
	"net"
	"sync"

	"github.com/vishvananda/netlink"
)

// RouteManager 路由管理器
type RouteManager struct {
	mu     sync.RWMutex
	groups map[string][]netlink.Route
}

// RouteConfig 路由配置结构体
type RouteConfig struct {
	Destination string // 目标网段，如 "192.168.1.0/24"
	Gateway     string // 网关，如 "192.168.1.1" (可选)
	Interface   string // 网络接口名，如 "eth0" (可选)
	Priority    int    // 路由优先级 (可选)
	Table       int    // 路由表 (可选)
}

// NewRouteManager 创建新的路由管理器
func NewRouteManager() *RouteManager {
	return &RouteManager{
		groups: make(map[string][]netlink.Route),
	}
}

// AddRoute 添加单条路由
func (rm *RouteManager) AddRoute(config *RouteConfig) error {
	route, err := rm.createNetlinkRoute(config)
	if err != nil {
		return err
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	if err := netlink.RouteAdd(route); err != nil {
		return fmt.Errorf("failed to add route: %v", err)
	}
	return nil
}

// DeleteRoute 删除单条路由
func (rm *RouteManager) DeleteRoute(config *RouteConfig) error {
	route, err := rm.createNetlinkRoute(config)
	if err != nil {
		return err
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	if err := netlink.RouteDel(route); err != nil {
		return fmt.Errorf("failed to delete route: %v", err)
	}

	// 从所有组中删除该路由
	for group, routes := range rm.groups {
		rm.groups[group] = removeRouteFromSlice(routes, *route)
	}

	return nil
}

// AddRouteToGroup 添加路由到组
func (rm *RouteManager) AddRouteToGroup(groupName string, config *RouteConfig) error {
	route, err := rm.createNetlinkRoute(config)
	if err != nil {
		return err
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	// 先添加到系统
	if err := netlink.RouteAdd(route); err != nil {
		return fmt.Errorf("failed to add route: %v", err)
	}

	// 添加到组
	rm.groups[groupName] = append(rm.groups[groupName], *route)
	return nil
}

// DeleteGroupRoutes 删除整个组的路由
func (rm *RouteManager) DeleteGroupRoutes(groupName string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	routes, exists := rm.groups[groupName]
	if !exists {
		return fmt.Errorf("group %s does not exist", groupName)
	}

	// 删除系统中的路由
	for _, route := range routes {
		if err := netlink.RouteDel(&route); err != nil {
			return fmt.Errorf("failed to delete route: %v", err)
		}
	}

	// 从组中删除
	delete(rm.groups, groupName)
	return nil
}

// ListGroupRoutes 列出组中的所有路由
func (rm *RouteManager) ListGroupRoutes(groupName string) ([]RouteConfig, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	routes, exists := rm.groups[groupName]
	if !exists {
		return nil, fmt.Errorf("group %s does not exist", groupName)
	}

	var configs []RouteConfig
	for _, route := range routes {
		configs = append(configs, rm.convertToRouteConfig(&route))
	}

	return configs, nil
}

// ListAllRoutes 列出所有路由
func (rm *RouteManager) ListAllRoutes() ([]RouteConfig, error) {
	routes, err := netlink.RouteList(nil, netlink.FAMILY_ALL)
	if err != nil {
		return nil, err
	}

	var configs []RouteConfig
	for _, route := range routes {
		configs = append(configs, rm.convertToRouteConfig(&route))
	}

	return configs, nil
}

// Helper methods

func (rm *RouteManager) createNetlinkRoute(config *RouteConfig) (*netlink.Route, error) {
	route := &netlink.Route{
		Priority: config.Priority,
		Table:    config.Table,
	}

	// 解析目标网段
	if config.Destination != "" {
		_, dst, err := net.ParseCIDR(config.Destination)
		if err != nil {
			return nil, fmt.Errorf("invalid destination CIDR: %v", err)
		}
		route.Dst = dst
	}

	// 解析网关
	if config.Gateway != "" {
		gw := net.ParseIP(config.Gateway)
		if gw == nil {
			return nil, fmt.Errorf("invalid gateway IP")
		}
		route.Gw = gw
	}

	// 解析接口
	if config.Interface != "" {
		link, err := netlink.LinkByName(config.Interface)
		if err != nil {
			return nil, fmt.Errorf("failed to find interface %s: %v", config.Interface, err)
		}
		route.LinkIndex = link.Attrs().Index
	}

	return route, nil
}

func (rm *RouteManager) convertToRouteConfig(route *netlink.Route) RouteConfig {
	config := RouteConfig{
		Priority: route.Priority,
		Table:    route.Table,
	}

	if route.Dst != nil {
		config.Destination = route.Dst.String()
	}

	if route.Gw != nil {
		config.Gateway = route.Gw.String()
	}

	if route.LinkIndex > 0 {
		if link, err := netlink.LinkByIndex(route.LinkIndex); err == nil {
			config.Interface = link.Attrs().Name
		}
	}

	return config
}

func removeRouteFromSlice(routes []netlink.Route, target netlink.Route) []netlink.Route {
	for i, route := range routes {
		if routesEqual(route, target) {
			return append(routes[:i], routes[i+1:]...)
		}
	}
	return routes
}

func routesEqual(a, b netlink.Route) bool {
	return a.Dst.String() == b.Dst.String() &&
		a.Gw.String() == b.Gw.String() &&
		a.Priority == b.Priority &&
		a.Table == b.Table &&
		a.LinkIndex == b.LinkIndex
}

// 更新某个组的所有路由，函数内对现有的路由和新的路由条目进行对比，把删除多出来的旧路由条目，添加多出来的新的路由条目
func (rm *RouteManager) UpdateGroupRoutes(groupName string, newRoutes []RouteConfig) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	needDel := []RouteConfig{}
	needAdd := []RouteConfig{}
	oldRoutes, _ := rm.ListGroupRoutes(groupName)

	// 记录旧路由中多出来的路由条目
	for _, old := range oldRoutes {
		if !containsRoute(newRoutes, old) {
			needDel = append(needDel, old)
		}
	}

	// 记录新路由中多出来的路由条目
	for _, newroute := range newRoutes {
		if !containsRoute(oldRoutes, newroute) {
			needAdd = append(needAdd, newroute)
		}
	}

	// 删除旧路由中多出来的路由条目
	for _, route := range needDel {
		if err := rm.AddRouteToGroup(groupName, &route); err != nil {
			return fmt.Errorf("failed to add route: %v", err)
		}
	}

	// 添加新的路由
	for _, route := range needAdd {
		if err := rm.AddRouteToGroup(groupName, &route); err != nil {
			return fmt.Errorf("failed to add route: %v", err)
		}
	}

	return nil
}

func containsRoute(routes []RouteConfig, config RouteConfig) bool {
	for _, route := range routes {
		if route == config {
			return true
		}
	}
	return false
}
