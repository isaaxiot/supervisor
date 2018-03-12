package supervisor

import (
	"errors"
)

const (
	updateFailed  = "Failed to update"
	startFailed   = "Failed to start"
	installFailed = "Failed to install"
	stopFailed    = "Failed to stop"
	removeFailed  = "Failed to remove"

	undefined = "undefined"
	running   = "running"
	stopped   = "stopped"
	removed   = "removed"
	installed = "installed"
	started   = "started"
	restarted = "restarted"
)

var (
	errNotInstalled     = errors.New("not installed")
	errNotRunning       = errors.New("service is not running")
	errOSNotSupported   = errors.New("OS not supported")
	errPermissionDenied = errors.New("permission denied")
	errAlreadyInstalled = errors.New("already installed")
)

// Service is supervised service
type Service interface {
	Status() (string, error)
	Restart() (string, error)
	Start() (string, error)
	Stop() (string, error)
	UpdateEnviron(env []string) (string, error)
	Install(args ...string) (string, error)
	Remove() (string, error)
	PID() (int, error)
	IsInstalled() bool
	ServiceName() string
}

// NewService returns new supervised service
func NewService(name, cmd, description, workingDir string, dependencies []string, environ map[string]string) Service {
	return newService(name, cmd, description, workingDir, dependencies, environ)
}

// GetSimple returns supervised instance
func GetSimple(name string) Service {
	return getService(name)
}
