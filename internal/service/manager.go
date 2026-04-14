package service

import "runtime"

func NewServiceManager() ServiceManager {
	switch runtime.GOOS {
	case "windows":
		return NewWindowsService()
	case "linux":
		return NewLinuxService()
	case "darwin":
		return NewMacOSService()
	default:
		panic("unsupported OS")
	}
}
