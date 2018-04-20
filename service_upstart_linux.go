package supervisor

import (
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

// upstart - standard record (struct) for linux upstart version of daemon package
type upstart struct {
	name         string
	cmd          string
	description  string
	dependencies []string
	workingDir   string
	logFile      string
	environ      map[string]string
}

// Standard service path for systemV daemons
func (u *upstart) servicePath() string {
	return "/etc/init/" + u.name + ".conf"
}

// Is a service installed
func (u *upstart) IsInstalled() bool {
	if _, err := os.Stat(u.servicePath()); err == nil {
		return true
	}
	return false
}

func (u *upstart) ServiceName() string {
	return u.name + ".conf"
}

// Check service is running
func (u *upstart) checkRunning() (int, error) {
	output, err := exec.Command("status", u.name).Output()
	if err == nil {
		if matched, err := regexp.MatchString(u.name+" start/running", string(output)); err == nil && matched {
			reg := regexp.MustCompile("process ([0-9]+)")
			data := reg.FindStringSubmatch(string(output))
			if len(data) > 1 {
				return strconv.Atoi(data[1])
			}
			return -1, nil
		}
	}
	return -1, errNotRunning
}

// Install the service
func (u *upstart) Install(args ...string) (string, error) {
	if ok, err := checkPrivileges(); !ok {
		return "", err
	}
	srvPath := u.servicePath()

	if u.IsInstalled() {
		return "", errAlreadyInstalled
	}
	file, err := os.Create(srvPath)

	if err != nil {
		return "", err
	}
	defer file.Close()

	templ, err := template.New("upstatConfig").Parse(upstatConfig)
	if err != nil {
		return "", err
	}
	if err := templ.Execute(
		file,
		&struct {
			Name, Description, Args, WorkingDir string
			Cmd                                 string
		}{
			Name:        u.name,
			Cmd:         u.cmd,
			Description: u.description,
			WorkingDir:  u.workingDir,
			Args:        strings.Join(args, " ")},
	); err != nil {
		return "", err
	}

	if err := os.Chmod(srvPath, 0755); err != nil {
		return "", err
	}
	return installed, nil
}

// Remove the service
func (u *upstart) Remove() (string, error) {
	if ok, err := checkPrivileges(); !ok {
		return "", err
	}
	if !u.IsInstalled() {
		return "", errNotInstalled
	}
	if err := os.Remove(u.servicePath()); err != nil {
		return "", err
	}
	return removed, nil
}

func (u *upstart) UpdateEnviron(env map[string]string) (string, error) {
	return "", nil
}

func (u *upstart) Restart() (string, error) {
	u.Stop()
	return u.Start()
}

func (u *upstart) PID() (int, error) {
	return u.checkRunning()
}

// Start the service
func (u *upstart) Start() (string, error) {
	if ok, err := checkPrivileges(); !ok {
		return "", err
	}
	if !u.IsInstalled() {
		return "", errNotInstalled
	}
	if err := exec.Command("start", u.name).Run(); err != nil {
		return "", err
	}
	return started, nil
}

// Stop the service
func (u *upstart) Stop() (string, error) {
	if ok, err := checkPrivileges(); !ok {
		return "", err
	}
	if !u.IsInstalled() {
		return "", errNotInstalled
	}
	if err := exec.Command("stop", u.name).Run(); err != nil {
		return "", err
	}
	return stopped, nil
}

// Status - Get service status
func (u *upstart) Status() (string, error) {
	if ok, err := checkPrivileges(); !ok {
		return "", err
	}
	if !u.IsInstalled() {
		return "Status could not defined", errNotInstalled
	}
	pid, err := u.checkRunning()
	if err != nil {
		return "", err
	}
	return "(pid: " + strconv.Itoa(pid) + ")", nil
}

var upstatConfig = `# {{.Name}} {{.Description}}

description     "{{.Description}}"

start on runlevel [2345]
stop on runlevel [016]

respawn
#kill timeout 5
chdir {{.WorkingDir}}
exec /bin/sh -c '{{.Cmd}} {{.Args}} >> {{.logFile}} 2>&1 '
`
