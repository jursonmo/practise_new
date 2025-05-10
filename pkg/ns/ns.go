package ns

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vishvananda/netns"
)

// 执行do函数，在ns ns中执行
func DoInNs(ns string, do func() error) error {
	if ns != "" {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		origin, err := netns.Get()
		if err != nil {
			return err
		}
		defer origin.Close()
		defer netns.Set(origin)

		newns, err := netns.GetFromName(ns)
		if err != nil {
			return err
		}
		err = netns.Set(newns)
		if err != nil {
			return err
		}
		defer newns.Close()
	}
	return do()
}

func ListNetNamespaces() ([]string, error) {
	var namespaces []string

	// 检查默认的/var/run/netns目录
	files, err := os.ReadDir("/var/run/netns")
	if err == nil {
		for _, f := range files {
			namespaces = append(namespaces, f.Name())
		}
		return namespaces, nil
	}

	// 如果/var/run/netns不存在，检查/proc/[pid]/ns/net
	procDirs, err := filepath.Glob("/proc/[0-9]*/ns/net")
	if err != nil {
		return nil, err
	}

	for _, procDir := range procDirs {
		pid := strings.Split(procDir, "/")[2]
		namespaces = append(namespaces, pid)
	}

	return namespaces, nil
}
