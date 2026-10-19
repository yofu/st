package st

import (
	"regexp"
	"strings"
	"testing"

	"github.com/yofu/st/arclm"
)

func TestBucklingDiagnosticOptionGrammar(t *testing.T) {
	pattern := regexp.MustCompile(bucklingNamedArgPattern)
	args := make(map[string]string)
	for _, token := range []string{"-diag=C:/temp/diag", "-diag-count=1", "-diag-shift=0.5,0.6", "-diag-only", "-diag-cpusec=30"} {
		m := pattern.FindStringSubmatch(token)
		if m == nil {
			t.Fatalf("diagnostic option became positional: %s", token)
		}
		args[strings.ToUpper(m[1])] = m[3]
	}
	options, err := arclm.ParseBucklingDiagnostics(args, "model.otp")
	if err != nil || options == nil || !options.CompareOnly || options.CompareLimit != 1 || len(options.CompareShifts) != 2 || options.CPUSeconds != 30 {
		t.Fatalf("options=%+v, err=%v", options, err)
	}
}
