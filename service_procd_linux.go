package supervisor

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

// procd - standard record (struct) for linux procd version of daemon package
type procd struct {
	name         string
	cmd          string
	description  string
	dependencies []string
	workingDir   string
	environ      map[string]string
}

// Standard service path for systemV daemons
func (u *procd) servicePath() string {
	return "/etc/init.d/" + u.name
}

// Is a service installed
func (u *procd) IsInstalled() bool {
	if _, err := os.Stat(u.servicePath()); err == nil {
		return true
	}
	return false
}

func (u *procd) ServiceName() string {
	return u.name
}

// Check service is running
func (u *procd) checkRunning() (int, error) {
	output, err := exec.Command(u.servicePath(), "status").Output()
	if err == nil {
		if matched, err := regexp.MatchString("running", string(output)); err == nil && matched {
			reg := regexp.MustCompile("running ([0-9]+)")
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
func (u *procd) Install(args ...string) (string, error) {
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

	var templ *template.Template
	if u.name == "isaax-agent" {
		templ, err = template.New("agentProcdConfig").Parse(agentProcdConfig)
	} else {
		templ, err = template.New("appProcdConfig").Parse(appProcdConfig)
	}
	if err != nil {
		return "", err
	}

	var env string
	if u.environ != nil {
		environ := mapToSlice(u.environ)
		env = environProcd(environ)
	}
	if err := templ.Execute(
		file,
		&struct {
			Name, Description, Args, WorkingDir string
			Cmd                                 string
			EnVar                               string
		}{
			Name:        u.name,
			Cmd:         u.cmd,
			Description: u.description,
			EnVar:       env,
			WorkingDir:  u.workingDir,
			Args:        strings.Join(args, " ")},
	); err != nil {
		return "", err
	}

	if err := os.Chmod(srvPath, 0755); err != nil {
		return "", err
	}
	file.Close()
	if u.name == "isaax-agent" {
		if err := exec.Command(u.servicePath(), "enable").Run(); err != nil {
			return "", err
		}
	}

	return installed, nil
}

// Remove the service
func (u *procd) Remove() (string, error) {
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

func (u *procd) UpdateEnviron(env []string) (string, error) {
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

	var templ *template.Template
	if u.name == "isaax-agent" {
		templ, err = template.New("agentProcdConfig").Parse(agentProcdConfig)
	} else {
		templ, err = template.New("appProcdConfig").Parse(appProcdConfig)
	}
	if err != nil {
		return "", err
	}

	if err := templ.Execute(
		file,
		&struct {
			Name, Description, Args, WorkingDir string
			Cmd                                 string
			EnVar                               string
		}{
			Name:        u.name,
			Cmd:         u.cmd,
			Description: u.description,
			EnVar:       environProcd(env),
			WorkingDir:  u.workingDir,
			Args:        strings.Join([]string{}, " ")},
	); err != nil {
		return "", err
	}

	if err := os.Chmod(srvPath, 0755); err != nil {
		return "", err
	}
	return "updated", nil
}

func (u *procd) Restart() (string, error) {
	if err := exec.Command(u.servicePath(), "restart").Run(); err != nil {
		return "", err
	}
	return "restarting", nil
}

func (u *procd) PID() (int, error) {
	return u.checkRunning()
}

// Start the service
func (u *procd) Start() (string, error) {
	if ok, err := checkPrivileges(); !ok {
		return "", err
	}
	if !u.IsInstalled() {
		return "", errNotInstalled
	}
	if err := exec.Command(u.servicePath(), "start").Run(); err != nil {
		return "", err
	}
	return started, nil
}

// Stop the service
func (u *procd) Stop() (string, error) {
	if ok, err := checkPrivileges(); !ok {
		return "", err
	}
	if !u.IsInstalled() {
		return "", errNotInstalled
	}
	if err := exec.Command(u.servicePath(), "stop").Run(); err != nil {
		return "", err
	}
	return stopped, nil
}

// Status - Get service status
func (u *procd) Status() (string, error) {
	if ok, err := checkPrivileges(); !ok {
		return "", err
	}
	if !u.IsInstalled() {
		return "Status could not be defined", errNotInstalled
	}
	pid, err := u.checkRunning()
	if err != nil {
		return "", err
	}
	return "(pid: " + strconv.Itoa(pid) + ")", nil
}

// MapToSlice converts map to slice in format k=v
func mapToSlice(m map[string]string) []string {
	v := make([]string, 0, len(m))
	for key, value := range m {
		v = append(v, fmt.Sprintf("%s=%s", key, value))
	}
	return v
}

// EnvironSystemd converts environ slice into systemd compatible environ
func environSystemd(environ []string) string {
	v := make([]string, 0, len(environ))
	for _, s := range environ {
		v = append(v, fmt.Sprintf("\"%s\"", s))
	}
	return strings.Join(v, " ")
}

// EnvironSystemd converts environ slice into systemd compatible environ
func environProcd(environ []string) string {
	v := make([]string, 0, len(environ))
	for _, s := range environ {
		v = append(v, fmt.Sprintf("\"%s\"", s))
	}
	env := strings.Join(v, " ")
	if env != "" {
		return "export " + env
	}
	return env

}

func checkPrivileges() (bool, error) {

	if output, err := exec.Command("id", "-g").Output(); err == nil {
		if gid, parseErr := strconv.ParseUint(strings.TrimSpace(string(output)), 10, 32); parseErr == nil {
			if gid == 0 {
				return true, nil
			}
			return false, errPermissionDenied
		}
	}
	return false, errOSNotSupported
}

var agentProcdConfig = `#!/bin/sh /etc/rc.common

# {{.Name}} {{.Description}}
USE_PROCD=1
START=120
STOP=120

start_service() {
  PROCD_DEBUG=1
  if [ -r /etc/init.d/isaax-project ]; then
    /etc/init.d/isaax-project start
  fi
  procd_open_instance
  procd_set_param command {{.Cmd}} {{.Args}}

  # respawn automatically if something died, be careful if you have an alternative process supervisor
  # if process dies sooner than respawn_threshold, it is considered crashed and after 5 retries the service is stopped
  procd_set_param respawn

  procd_set_param limits core="unlimited"  # If you need to set ulimit for your process
  procd_close_instance
}
`
var appProcdConfig = `#!/bin/sh

dir="{{.WorkingDir}}"
cmd="{{.Cmd}} {{.Args}}"
user="root"
{{.EnVar}}
name="{{.Name}}"
pid_file="/var/run/$name.pid"
stdout_log="/var/log/$name.log"
stderr_log="/var/log/$name.err"

get_pid() {
    cat "$pid_file"
}

is_running() {
    [ -f "$pid_file" ] && kill -0 $(get_pid) > /dev/null 2>&1
}

case "$1" in
    start)
    if is_running; then
        echo "Already started"
    else
        echo "Starting $name"
        cd "$dir"
        $cmd >> "$stdout_log" 2>> "$stderr_log" &
        echo $! > "$pid_file"
        if ! is_running; then
            echo "Unable to start, see $stdout_log and $stderr_log"
            exit 1
        fi
    fi
    ;;
    stop)
    if is_running; then
        echo -n "Stopping $name.."
        kill $(get_pid)
        for i in 1 2 3 4 5 6 7 8 9 10
        # for i in $(seq 10)
        do
            if ! is_running; then
                break
            fi

            echo -n "."
            sleep 1
        done
        echo

        if is_running; then
            echo "Not stopped; may still be shutting down or shutdown may have failed"
            exit 1
        else
            echo "Stopped"
            if [ -f "$pid_file" ]; then
                rm "$pid_file"
            fi
        fi
    else
        echo "Not running"
    fi
    ;;
    restart)
    $0 stop
    if is_running; then
        echo "Unable to stop, will not attempt to start"
        exit 1
    fi
    $0 start
    ;;
    status)
    if is_running; then
        echo "Running "$(get_pid) 
    else
        echo "Stopped"
        exit 1
    fi
    ;;
    *)
    echo "Usage: $0 {start|stop|restart|status}"
    exit 1
    ;;
esac

exit 0`
