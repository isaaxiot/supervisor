package supervisor

import (
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

type systemD struct {
	name         string
	cmd          string
	description  string
	dependencies []string
	workingDir   string
	logFile      string
	environ      map[string]string
	restart      string
	restartSec   string
}

func (s *systemD) Status() (string, error) {
	if ok, err := isRoot(); !ok {
		return undefined, err
	}
	if !s.IsInstalled() {
		return undefined, errNotInstalled
	}
	if pid, r := s.isRunning(); r {
		return running + "(pid: " + strconv.Itoa(pid) + ")", nil
	}
	return stopped, nil
}

func (s *systemD) Restart() (string, error) {
	if err := exec.Command("systemctl", "restart", s.ServiceName()).Run(); err != nil {
		return startFailed, err
	}
	return "restarting", nil
}

func (s *systemD) UpdateEnviron(env map[string]string) (string, error) {

	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return updateFailed, err
	}

	if err := exec.Command("systemctl", "enable", s.ServiceName()).Run(); err != nil {
		return updateFailed, err
	}

	return "updated", nil
}

func (s *systemD) Start() (string, error) {
	//check if calling user is root
	if ok, err := isRoot(); !ok {
		return startFailed, err
	}
	//start app via systemctl
	if err := exec.Command("systemctl", "start", s.ServiceName()).Run(); err != nil {
		return startFailed, err
	}

	return "starting", nil
}

func (s *systemD) Stop() (string, error) {
	//check if calling user is root
	if ok, err := isRoot(); !ok {
		return stopFailed, err
	}
	//start app via systemctl
	if err := exec.Command("systemctl", "stop", s.ServiceName()).Run(); err != nil {
		return stopFailed, err
	}

	return "stopping", nil
}

func (s *systemD) Install(args ...string) (string, error) {
	//check if calling user is root
	if ok, err := isRoot(); !ok {
		return "Failed to install ", err
	}
	//check if app has a service unit file
	if s.IsInstalled() {
		return installFailed, errAlreadyInstalled
	}

	file, err := os.Create(s.unitFile())
	if err != nil {
		return installFailed, err
	}
	defer file.Close()

	t, err := template.New("systemdConfig").Parse(systemDConfig)
	if err != nil {
		return installFailed, err
	}
	var env string
	if s.environ != nil {
		environ := mapToSlice(s.environ)
		env = environSystemd(environ)
	}
	if err := t.Execute(
		file,
		&struct {
			Name         string
			Cmd          string
			Description  string
			Dependencies string
			Args         string
			LogFile      string
			EnVar        string
			EnvFile      string
			Restart      string
			RestartSec   string
			WorkingDir   string
		}{
			Name:         s.name,
			Cmd:          s.cmd,
			Description:  s.description,
			Dependencies: strings.Join(s.dependencies, " "),
			Args:         strings.Join(args, " "),
			EnVar:        env,
			EnvFile:      path.Join(s.workingDir, s.name+".env"),
			Restart:      s.restart,
			WorkingDir:   s.workingDir,
			LogFile:      s.logFile,
			RestartSec:   s.restartSec,
		},
	); err != nil {
		return installFailed, err
	}

	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return installFailed, err
	}

	if err := exec.Command("systemctl", "enable", s.ServiceName()).Run(); err != nil {
		return installFailed, err
	}

	return "installed", nil
}

func (s *systemD) Remove() (string, error) {
	//check if calling user is root
	if ok, err := isRoot(); !ok {
		return removeFailed, err
	}
	//check if app has a service unit file
	if !s.IsInstalled() {
		return removeFailed, errNotInstalled
	}

	if err := exec.Command("systemctl", "disable", s.ServiceName()).Run(); err != nil {
		return removeFailed, err
	}

	if err := os.Remove(s.unitFile()); err != nil {
		return removeFailed, err
	}

	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return updateFailed, err
	}

	return removed, nil
}

func (s *systemD) PID() (int, error) {
	return s.pid()
}

func (s *systemD) unitFile() string {
	return "/etc/systemd/system/" + s.name + ".service"
}

func (s *systemD) ServiceName() string {
	return s.name + ".service"
}

func (s *systemD) IsInstalled() bool {
	if _, err := os.Stat(s.unitFile()); err == nil {
		return true
	}
	return false
}

func (s *systemD) isRunning() (int, bool) {
	pid, _ := s.pid()
	if pid > 0 {
		return pid, true
	}
	return pid, false
}

func (s *systemD) pid() (int, error) {
	out, err := exec.Command("systemctl", "status", s.ServiceName()).Output()
	if err == nil {
		if matched, err := regexp.MatchString("Active: active", string(out)); err == nil && matched {
			reg := regexp.MustCompile("Main PID: ([0-9]+)")
			data := reg.FindStringSubmatch(string(out))
			if len(data) > 1 {
				return strconv.Atoi(data[1])
			}
			return -1, nil
		}
	}
	return -1, errNotRunning
}

func isRoot() (bool, error) {
	if output, err := exec.Command("id", "-g").Output(); err == nil {
		if gid, err := strconv.ParseUint(strings.TrimSpace(string(output)), 10, 32); err == nil {
			if gid == 0 {
				return true, nil
			}
			return false, errPermissionDenied
		}
	}
	return false, errOSNotSupported
}

var systemDConfig = `[Unit]
Description={{.Description}}
Requires={{.Dependencies}}
After={{.Dependencies}}

[Service]
CPUAccounting=yes
MemoryAccounting=yes
PIDFile=/var/run/{{.Name}}.pid
ExecStartPre=/bin/rm -f /var/run/{{.Name}}.pid
ExecStart=/bin/sh -c '{{.Cmd}} {{.Args}} >>{{.LogFile}} 2>&1'
WorkingDirectory={{.WorkingDir}}
Environment={{.EnVar}}
EnvironmentFile={{.EnvFile}}
Restart={{.Restart}}
RestartSec={{.RestartSec}}

[Install]
WantedBy=multi-user.target
`
