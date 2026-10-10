package plot

import (
	"os"
	"testing"
)

// Set GOPLOT_VERIFY_DIR to write the rendered test figures somewhere to look
// at them (outside the repository).
func dumpFigure(t *testing.T, name string, png, svg []byte) {
	dir := os.Getenv("GOPLOT_VERIFY_DIR")
	if dir == "" {
		return
	}
	_ = os.WriteFile(dir+"/test_"+name+".png", png, 0o644)
	_ = os.WriteFile(dir+"/test_"+name+".svg", svg, 0o644)
}
