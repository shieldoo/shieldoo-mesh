package inputbox

import (
	"os/exec"
	"strings"
)

// InputBox displays a dialog box, returning the entered value and a bool for success
func InputBox(title, message, defaultAnswer string) (string, bool) {
	out, err := exec.Command(
		"zenity", "--entry",
		"--title", title,
		"--text", message,
		"--entry-text", defaultAnswer).Output()
	// NOTE: exit code 1 = cancel was pressed
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}
