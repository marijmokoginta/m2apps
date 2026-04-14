package service

type ServiceManager interface {
	Install() error
	Start() error
	Stop() error
	Status() (string, error)
}
