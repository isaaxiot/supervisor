package supervisor

import (
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"strconv"
	"text/template"
	"time"
)

// darwin - standard record (struct) for darwin version of daemon package
type darwin struct {
	name         string
	cmd          string
	description  string
	dependencies []string
	workingDir   string
	restart      string
	restartSec   string
	logDir       string
}

func newService(name, cmd, description, workingDir string, dependencies []string, environ map[string]string) Service {
	return &darwin{name: name, cmd: cmd, description: description, workingDir: workingDir, dependencies: dependencies}
}

func getService(name string) Service {
	return &darwin{name: name}
}

// Standard service path for system daemons
func (d *darwin) servicePath() string {
	u, err := user.Current()
	if err != nil {
		return "/Library/LaunchDaemons/" + d.ServiceName()
	}
	return u.HomeDir + "/Library/LaunchAgents/" + d.ServiceName()
}

func (d *darwin) ServiceName() string {
	return d.name + ".plist"
}

func (d *darwin) UpdateEnviron(env []string) (string, error) {
	return "", nil
}

// Is a service installed
func (d *darwin) IsInstalled() bool {
	if _, err := os.Stat(d.servicePath()); err == nil {
		return true
	}
	return false
}

func (d *darwin) PID() (int, error) {
	if pid, running := d.checkRunning(); running {
		return strconv.Atoi(pid)
	}
	return -1, nil
}

// Check service is running
func (d *darwin) checkRunning() (string, bool) {
	output, err := exec.Command("launchctl", "list", d.name).Output()
	if err == nil {
		if matched, err := regexp.MatchString(d.name, string(output)); err == nil && matched {
			reg := regexp.MustCompile("PID\" = ([0-9]+);")
			data := reg.FindStringSubmatch(string(output))
			if len(data) > 1 {
				return data[1], true
			}
			return "0", true
		}
	}
	return "-1", false
}

// Install the service
func (d *darwin) Install(args ...string) (string, error) {
	srvPath := d.servicePath()
	if d.IsInstalled() {
		return installFailed, errAlreadyInstalled
	}

	file, err := os.Create(srvPath)
	if err != nil {
		return installFailed, err
	}
	defer file.Close()

	templ, err := template.New("propertyList").Parse(propertyList)
	if err != nil {
		return installFailed, err
	}

	if err := templ.Execute(
		file,
		&struct {
			Name, Cmd          string
			WorkingDir, LogDir string
			Args               []string
		}{
			Name: d.name, Cmd: d.cmd, Args: args, WorkingDir: d.workingDir, LogDir: d.logDir,
		},
	); err != nil {
		return installFailed, err
	}

	return installed, nil
}

// Remove the service
func (d *darwin) Remove() (string, error) {
	if !d.IsInstalled() {
		return "", errNotInstalled
	}

	if err := run("launchctl", "remove", d.name); err != nil {
		return "", err
	}

	if err := os.Remove(d.servicePath()); err != nil {
		return "", err
	}
	return removed, nil
}

// Start the service
func (d *darwin) Start() (string, error) {
	if !d.IsInstalled() {
		return "", errNotInstalled
	}

	if err := run("launchctl", "load", d.servicePath()); err != nil {
		return "", err
	}
	return started, nil
}

// Stop the service
func (d *darwin) Stop() (string, error) {
	if !d.IsInstalled() {
		return "", errNotInstalled
	}

	if err := run("launchctl", "unload", d.servicePath()); err != nil {
		return "", err
	}
	return stopped, nil
}

func (d *darwin) Restart() (string, error) {
	if s, err := d.Stop(); err != nil {
		return s, err
	}

	time.Sleep(50 * time.Millisecond)
	if s, err := d.Start(); err != nil {
		return s, err
	}
	return "restarted", nil
}

// Status - Get service status
func (d *darwin) Status() (string, error) {
	if !d.IsInstalled() {
		return "", errNotInstalled
	}

	statusAction, _ := d.checkRunning()
	return statusAction, nil
}

var propertyList = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key><string>{{html .Name}}</string>
	<key>ProgramArguments</key>
	<array>
		<string>{{.Cmd}}</string>
{{range .Args}}
        <string>{{html .}}</string>
{{end}}
	</array>
    <key>WorkingDirectory</key>
    <string>{{.WorkingDir}}</string>

    <key>StandardErrorPath</key>
    <string>{{.LogDir}}{{.Name}}.log</string>
    <key>StandardOutPath</key>
    <string>{{.LogDir}}{{.Name}}.log</string>

    <key>SessionCreate</key>
    <false/>
	<key>KeepAlive</key>
	<true/>
	<key>RunAtLoad</key>
	<true/>
    <key>Disabled</key>
    <false/>
</dict>
</plist>
`
