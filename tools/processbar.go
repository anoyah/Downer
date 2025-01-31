package tools

import (
	"fmt"
	"os"
	"strings"
)

// ProgressBar download process bar
func ProgressBar(current, total int, desc string) {
	if total < current {
		return
	}

	percentage := float64(current) / float64(total) * 100
	barLength := 20
	arrow := ">"

	if current == total {
		arrow = ""
		barLength += 1
	}

	filledLength := int(barLength * current / total)

	completed := strings.Repeat("=", filledLength)
	restOf := strings.Repeat(" ", barLength-filledLength)
	bar := fmt.Sprintf("[%s%s%s]", completed, arrow, restOf)

	std := os.Stdout

	fmt.Fprintf(std, "\r%s: %s %.2f%%", desc, bar, percentage)
	if current == total {
		fmt.Fprintln(std)
	}
}
