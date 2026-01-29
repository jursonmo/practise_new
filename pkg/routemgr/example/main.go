package main

import (
	"fmt"

	router "github.com/jursonmo/practise_new/pkg/routemgr" // 替换为你的实际包路径
)

func main() {
	rm := router.NewRouteManager()

	// 示例1: 添加单条路由(通过网段和接口)
	err := rm.AddRoute(&router.RouteConfig{
		Destination: "192.168.1.0/24",
		Interface:   "eth0",
	})
	if err != nil {
		fmt.Printf("Error adding route: %v\n", err)
	}

	// 示例2: 添加单条路由(通过网段和网关)
	err = rm.AddRoute(&router.RouteConfig{
		Destination: "10.10.10.0/24",
		Gateway:     "192.168.1.1",
		Priority:    100,
	})
	if err != nil {
		fmt.Printf("Error adding route: %v\n", err)
	}

	// 示例3: 添加路由到组
	err = rm.AddRouteToGroup("vpn_routes", &router.RouteConfig{
		Destination: "172.16.1.1/24",
		Gateway:     "10.8.0.1",
		Interface:   "tun0",
	})
	if err != nil {
		fmt.Printf("Error adding route to group: %v\n", err)
	}

	// 示例4: 列出组路由
	routes, err := rm.ListGroupRoutes("vpn_routes")
	if err != nil {
		fmt.Printf("Error listing group routes: %v\n", err)
	} else {
		fmt.Println("Routes in vpn_routes group:")
		for _, r := range routes {
			fmt.Printf("- Destination: %s, Gateway: %s, Interface: %s\n",
				r.Destination, r.Gateway, r.Interface)
		}
	}

	// 示例5: 删除整个组的路由
	err = rm.DeleteGroupRoutes("vpn_routes")
	if err != nil {
		fmt.Printf("Error deleting group routes: %v\n", err)
	}

	// 示例6: 列出所有路由
	allRoutes, err := rm.ListAllRoutes()
	if err != nil {
		fmt.Printf("Error listing all routes: %v\n", err)
	} else {
		fmt.Println("All routes:")
		for _, r := range allRoutes {
			fmt.Printf("- %s via %s (dev %s)\n",
				r.Destination, r.Gateway, r.Interface)
		}
	}
}
