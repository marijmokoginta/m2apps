package service

import "fmt"

type WindowsService struct{}

func NewWindowsService() ServiceManager {
	return &WindowsService{}
}

func (s *WindowsService) Install() error {
	fmt.Println("Windows service not implemented yet")
	return nil
}

func (s *WindowsService) Start() error {
	fmt.Println("Windows service not implemented yet")
	return nil
}

func (s *WindowsService) Stop() error {
	fmt.Println("Windows service not implemented yet")
	return nil
}

func (s *WindowsService) Status() (string, error) {
	return "Windows service not implemented yet", nil
}
