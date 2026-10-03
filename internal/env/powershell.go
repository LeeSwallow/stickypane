package env

import (
	"fmt"
	"path/filepath"
	"strings"
)

// What is PowerShell's own is kept here: how to tell it is the shell, how
// it sets a variable and how it runs a script.

// inPowerShell reports whether PSModulePath is the one a PowerShell session
// sets. The variable exists system-wide on Windows, so its presence says
// nothing; a session adds the user's own module folder to it, which a
// command prompt does not have.
func inPowerShell(psModulePath string) bool {
	for _, p := range strings.Split(psModulePath, ";") {
		p = strings.ToLower(strings.ReplaceAll(p, `\`, "/"))
		if strings.Contains(p, "/documents/") && (strings.HasSuffix(p, "powershell/modules") || strings.HasSuffix(p, "windowspowershell/modules")) {
			return true
		}
	}
	return false
}

func powerShellSetVar(name, value string) string {
	return fmt.Sprintf(`$env:%s = "%s"`, name, strings.ReplaceAll(value, `"`, "`\""))
}

// powerShellScript runs a .ps1 with PowerShell 7 (pwsh) when it is there,
// else Windows PowerShell, without the user's profile and without the
// execution policy stopping a file the user just said yes to.
func (e Env) powerShellScript(path string) (string, []string, error) {
	for _, exe := range []string{"pwsh", "powershell"} {
		if _, err := e.lookPath(exe); err == nil {
			return exe, []string{"-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", path}, nil
		}
	}
	return "", nil, fmt.Errorf("running %s needs PowerShell (pwsh or powershell)", filepath.Base(path))
}

// powerShellCommand runs a command line in the PowerShell that is the
// user's shell, without their profile.
func powerShellCommand(name, line string) (string, []string) {
	return name, []string{"-NoProfile", "-Command", line}
}
