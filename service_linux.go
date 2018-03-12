package supervisor

import (
	"os"
	"strings"
)

// newService returns new supervised service
func newService(name, cmd, description, workingDir string, dependencies []string, environ map[string]string) Service {
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		return newSystemDService(name, cmd, description, workingDir, dependencies, environ)
	}

	if _, err := os.Stat("/sbin/initctl"); err == nil {
		return newUpstartService(name, cmd, description, workingDir, dependencies, environ)
	}

	if _, err := os.Stat("/sbin/procd"); err == nil {
		return newProcDService(name, cmd, description, workingDir, dependencies, environ)
	}

	return newSystemVService(name, cmd, description, workingDir, dependencies, environ)
}

func getService(name string) Service {
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		return &systemD{name: name}
	}

	if _, err := os.Stat("/sbin/initctl"); err == nil {
		return &upstart{name: name}
	}

	if _, err := os.Stat("/sbin/procd"); err == nil {
		return &procd{name: name}
	}

	return &systemV{name: name}
}

func newSystemDService(name, cmd, description, workingDir string, dependencies []string, environ map[string]string) Service {
	return &systemD{
		name:         name,
		cmd:          cmd,
		workingDir:   workingDir,
		restart:      "on-failure",
		restartSec:   "10",
		environ:      environ,
		description:  description,
		dependencies: dependencies,
	}
}

func newSystemVService(name, cmd, description, workingDir string, dependencies []string, environ map[string]string) Service {
	return &systemV{
		name:         name,
		cmd:          cmd,
		workingDir:   workingDir,
		environ:      environ,
		description:  description,
		dependencies: dependencies,
	}
}

func newUpstartService(name, cmd, description, workingDir string, dependencies []string, environ map[string]string) Service {
	return &upstart{
		name:         name,
		cmd:          cmd,
		workingDir:   workingDir,
		environ:      environ,
		description:  description,
		dependencies: dependencies,
	}
}

func newProcDService(name, cmd, description, workingDir string, dependencies []string, environ map[string]string) Service {
	return &procd{
		name:         name,
		cmd:          cmd,
		workingDir:   workingDir,
		environ:      environ,
		description:  description,
		dependencies: dependencies,
	}
}

func legacyUnitFile(name string) Service {
	name = strings.Replace(name, " ", "_", -1)
	return &systemD{name: name}
}
