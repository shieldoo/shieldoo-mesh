package inputbox

import (
	"os/exec"
	"strings"
)

// InputBox displays a dialog box, returning the entered value and a bool for success
func InputBox(title, message, defaultAnswer string) (string, bool) {
	out, err := exec.Command(
		"osascript",
		"-e",
		`set T to text returned of (display dialog "`+
			message+`" buttons {"Cancel", "OK"} default button "OK" with title "`+title+`" default answer "`+
			defaultAnswer+`")`).Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}
