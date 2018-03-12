// +build linux darwin

package supervisor

import (
	"fmt"
	"io/ioutil"
	"os/exec"
)

func run(command string, arguments ...string) error {
	cmd := exec.Command(command, arguments...)

	// Connect pipe to read Stderr
	stderr, err := cmd.StderrPipe()

	if err != nil {
		// Failed to connect pipe
		return fmt.Errorf("%q failed to connect stderr pipe: %v", command, err)
	}

	// Do not use cmd.Run()
	if err := cmd.Start(); err != nil {
		// Problem while copying stdin, stdout, or stderr
		return fmt.Errorf("%q failed: %v", command, err)
	}

	// Zero exit status
	// Darwin: launchctl can fail with a zero exit status,
	// so check for emtpy stderr
	if command == "launchctl" {
		slurp, _ := ioutil.ReadAll(stderr)
		if len(slurp) > 0 {
			return fmt.Errorf("%q failed with stderr: %s", command, slurp)
		}
	}

	if err := cmd.Wait(); err != nil {
		// Command didn't exit with a zero exit status.
		return fmt.Errorf("%q failed: %v", command, err)
	}

	return nil
}

func runWithOutput(command string, arguments ...string) ([]byte, error) {
	cmd := exec.Command(command, arguments...)
	return cmd.CombinedOutput()
}
