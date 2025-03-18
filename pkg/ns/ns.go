package ns

import (
	"runtime"

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
