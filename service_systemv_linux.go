package supervisor

import (
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

// systemV - standard record (struct) for linux systemV version of daemon package
type systemV struct {
	name         string
	cmd          string
	description  string
	dependencies []string
	workingDir   string
	logFile      string
	environ      map[string]string
}

// Standard service path for systemV daemons
func (l *systemV) servicePath() string {
	return "/etc/init.d/" + l.name
}

// Is a service installed
func (l *systemV) IsInstalled() bool {
	if _, err := os.Stat(l.servicePath()); err == nil {
		return true
	}
	return false
}

// Check service is running
func (l *systemV) checkRunning() (int, error) {
	output, err := exec.Command("service", l.name, "status").Output()
	if err == nil {
		if matched, err := regexp.MatchString(l.name, string(output)); err == nil && matched {
			reg := regexp.MustCompile("pid  ([0-9]+)")
			data := reg.FindStringSubmatch(string(output))
			if len(data) > 1 {
				return strconv.Atoi(data[1])
			}
			return -1, nil
		}
	}
	return -1, errNotRunning
}

func (l *systemV) PID() (int, error) {
	return l.checkRunning()
}

// Install the service
func (l *systemV) Install(args ...string) (string, error) {
	if ok, err := checkPrivileges(); !ok {
		return "", err
	}

	if l.IsInstalled() {
		return "", errAlreadyInstalled
	}

	file, err := os.Create(l.servicePath())
	if err != nil {
		return "", err
	}
	defer file.Close()

	templ, err := template.New("systemVConfig").Parse(systemVConfig)
	if err != nil {
		return "", err
	}

	if err := templ.Execute(
		file,
		&struct {
			Name, Description   string
			WorkingDir, LogFile string
			Args, Cmd           string
		}{
			Name:        l.name,
			Cmd:         l.cmd,
			WorkingDir:  l.workingDir,
			LogFile:     l.logFile,
			Description: l.description,
			Args:        strings.Join(args, " "),
		},
	); err != nil {
		return "", err
	}

	if err := os.Chmod(l.servicePath(), 0755); err != nil {
		return "", err
	}

	for _, i := range [...]string{"2", "3", "4", "5"} {
		if err := os.Symlink(l.servicePath(), "/etc/rc"+i+".d/S87"+l.name); err != nil {
			continue
		}
	}
	for _, i := range [...]string{"0", "1", "6"} {
		if err := os.Symlink(l.servicePath(), "/etc/rc"+i+".d/K17"+l.name); err != nil {
			continue
		}
	}

	return installed, nil
}

func (l *systemV) ServiceName() string {
	return l.name
}

// Remove the service
func (l *systemV) Remove() (string, error) {
	if ok, err := checkPrivileges(); !ok {
		return "", err
	}

	if !l.IsInstalled() {
		return "", errNotInstalled
	}

	if err := os.Remove(l.servicePath()); err != nil {
		return "", err
	}

	for _, i := range [...]string{"2", "3", "4", "5"} {
		if err := os.Remove("/etc/rc" + i + ".d/S87" + l.name); err != nil {
			continue
		}
	}
	for _, i := range [...]string{"0", "1", "6"} {
		if err := os.Remove("/etc/rc" + i + ".d/K17" + l.name); err != nil {
			continue
		}
	}

	return removed, nil
}

func (l *systemV) isRunning() (int, bool) {
	pid, err := l.checkRunning()
	if err != nil {
		return -1, false
	}
	if pid > 0 {
		return pid, true
	}
	return pid, false
}

// Start the service
func (l *systemV) Start() (string, error) {
	if ok, err := checkPrivileges(); !ok {
		return "", err
	}
	if !l.IsInstalled() {
		return "", errNotInstalled
	}
	if err := exec.Command("service", l.name, "start").Run(); err != nil {
		return "", err
	}
	return started, nil
}

func (l *systemV) Restart() (string, error) {
	if err := exec.Command("service", l.name, "restart").Run(); err != nil {
		return "", err
	}
	return restarted, nil
}

func (l *systemV) UpdateEnviron(environ map[string]string) (string, error) {
	return "", nil
}

// Stop the service
func (l *systemV) Stop() (string, error) {
	if ok, err := checkPrivileges(); !ok {
		return "", err
	}
	if !l.IsInstalled() {
		return "", errNotInstalled
	}
	if err := exec.Command("service", l.name, "stop").Run(); err != nil {
		return "", err
	}
	return stopped, nil
}

// Status - Get service status
func (l *systemV) Status() (string, error) {

	if ok, err := isRoot(); !ok {
		return undefined, err
	}
	if !l.IsInstalled() {
		return undefined, errNotInstalled
	}
	if pid, r := l.isRunning(); r {
		return running + "(pid: " + strconv.Itoa(pid) + ")", nil
	}
	return stopped, nil
}

var systemVConfig = `#! /bin/sh
#
#       /etc/rc.d/init.d/{{.Name}}
#
#       Starts {{.Name}} as a daemon
#
# chkconfig: 2345 87 17
# description: Starts and stops a single {{.Name}} instance on this system

### BEGIN INIT INFO
# Provides: {{.Name}} 
# Required-Start: $network $named
# Required-Stop: $network $named
# Default-Start: 2 3 4 5
# Default-Stop: 0 1 6
# Short-Description: This service manages the {{.Description}}.
# Description: {{.Description}}
### END INIT INFO

#
# Source function library.
#
if [ -f /etc/rc.d/init.d/functions ]; then
    . /etc/rc.d/init.d/functions
fi

proc="{{.Name}}"
pidfile="/var/run/$proc.pid"
lockfile="/var/lock/subsys/$proc"
stdoutlog="$logFile"
stderrlog="$logFile"

exec="/bin/bash -c '{{.Cmd}} {{.Args}} >> $stdoutlog 2>> $stderrlog & ' "
servname="{{.Description}}"


[ -d $(dirname $lockfile) ] || mkdir -p $(dirname $lockfile)

[ -e /etc/sysconfig/$proc ] && . /etc/sysconfig/$proc

start() {
	cd {{.WorkingDir}}
    [ -x $(exec) ] || exit 5

    if [ -f $pidfile ]; then
        if ! [ -d "/proc/$(cat $pidfile)" ]; then
            rm $pidfile
            if [ -f $lockfile ]; then
                rm $lockfile
            fi
        fi
    fi

    if ! [ -f $pidfile ]; then
        printf "Starting $servname:\t"
        echo "$(date)" >> $stdoutlog
        $(exec)
        echo $! > $pidfile
        touch $lockfile
        success
        echo
    else
        # failure
        echo
        printf "$pidfile still exists...\n"
        exit 7
    fi
}

stop() {
    echo -n $"Stopping $servname: "
    killproc -p $pidfile $proc
    retval=$?
    echo
    [ $retval -eq 0 ] && rm -f $lockfile
    return $retval
}

restart() {
    stop
    start
}

rh_status() {
    status -p $pidfile $proc
}

rh_status_q() {
    rh_status >/dev/null 2>&1
}

case "$1" in
    start)
        rh_status_q && exit 0
        $1
        ;;
    stop)
        rh_status_q || exit 0
        $1
        ;;
    restart)
        $1
        ;;
    status)
        rh_status
        ;;
    *)
        echo $"Usage: $0 {start|stop|status|restart}"
        exit 2
esac

exit $?
`
