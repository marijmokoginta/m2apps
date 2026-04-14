package service

import "fmt"

type LinuxService struct{}

func NewLinuxService() ServiceManager {
	return &LinuxService{}
}

func (s *LinuxService) Install() error {
	fmt.Println("Linux systemd service not implemented yet")
	return nil
}

func (s *LinuxService) Start() error {
	fmt.Println("Linux systemd service not implemented yet")
	return nil
}

func (s *LinuxService) Stop() error {
	fmt.Println("Linux systemd service not implemented yet")
	return nil
}

func (s *LinuxService) Status() (string, error) {
	return "Linux systemd service not implemented yet", nil
}
