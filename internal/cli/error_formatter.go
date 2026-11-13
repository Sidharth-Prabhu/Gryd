package cli

import (
	"fmt"
)

// FormatGrydError formats errors as clean terminal text with Gryd branding.
func FormatGrydError(err error) string {
	if err == nil {
		return ""
	}

	return fmt.Sprintf("\033[1;31m[gryd:error]\033[0m %s\n", err.Error())
}
