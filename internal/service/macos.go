package service

import "fmt"

type MacOSService struct{}

func NewMacOSService() ServiceManager {
	return &MacOSService{}
}

func (s *MacOSService) Install() error {
	fmt.Println("macOS launchd service not implemented yet")
	return nil
}

func (s *MacOSService) Start() error {
	fmt.Println("macOS launchd service not implemented yet")
	return nil
}

func (s *MacOSService) Stop() error {
	fmt.Println("macOS launchd service not implemented yet")
	return nil
}

func (s *MacOSService) Status() (string, error) {
	return "macOS launchd service not implemented yet", nil
}
