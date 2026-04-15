package service

type ServiceManager interface {
	Install() error
	Uninstall() error
	Enable() error
	Disable() error
	Start() error
	Stop() error
	Status() (string, error)
}
