package st

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/yofu/st/arclm"
)

// prepareArclm101 validates the command before copying any analysis state.
// Display periods contain result indices, while Arclms holds the latest state
// under the base period only. Older displayed steps are not material snapshots.
func prepareArclm101(frame *Frame, args map[string]string, output string) (string, *arclm.Frame, *arclm.AnalysisCondition, int, error) {
	cond := arclm.NewAnalysisCondition()
	_, init := args["INIT"]
	if _, noinit := args["NOINIT"]; init && noinit {
		return "", nil, nil, 0, fmt.Errorf(":arclm101: -init and -noinit cannot be combined")
	}
	cond.SetInit(init)
	display := strings.Split(strings.ToUpper(frame.Show.Period), "@")
	per := display[0]
	if p := args["PERIOD"]; p != "" {
		per = strings.ToUpper(p)
	}
	af := frame.Arclms[per]
	if af == nil {
		return "", nil, nil, 0, fmt.Errorf(":arclm101: frame isn't extracted to period %s", per)
	}
	if af.Running() {
		return "", nil, nil, 0, fmt.Errorf("analysis is running")
	}
	if s, ok := args["STEP"]; ok {
		parts := strings.Split(s, ";")
		if len(parts) > 4 {
			return "", nil, nil, 0, fmt.Errorf(":arclm101: use -step=nlap;delta[;start;max]")
		}
		keys := []string{"NLAP", "DELTA", "START", "MAX"}
		// Named options override the corresponding legacy STEP field.
		copyArgs := make(map[string]string, len(args)+4)
		for k, v := range args {
			copyArgs[k] = v
		}
		for i, value := range parts {
			if _, exists := copyArgs[keys[i]]; !exists && value != "" {
				copyArgs[keys[i]] = value
			}
		}
		args = copyArgs
	}
	if s, ok := args["NLAP"]; ok {
		n, err := strconv.Atoi(s)
		if err != nil || n <= 0 {
			return "", nil, nil, 0, fmt.Errorf(":arclm101: NLAP must be a positive integer")
		}
		cond.SetNlap(n)
	}
	for _, option := range []struct {
		key string
		set func(float64)
	}{
		{"DELTA", cond.SetDelta}, {"START", cond.SetStart}, {"MAX", cond.SetMax}, {"EPS", cond.SetEps},
	} {
		if s, ok := args[option.key]; ok {
			v, err := strconv.ParseFloat(s, 64)
			if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || (option.key == "DELTA" && v == 0) || (option.key == "EPS" && v <= 0) {
				return "", nil, nil, 0, fmt.Errorf(":arclm101: invalid %s=%s", option.key, s)
			}
			option.set(v)
		}
	}
	offset := 0
	if !init {
		source := frame.Arclms[display[0]]
		if source == nil {
			return "", nil, nil, 0, fmt.Errorf(":arclm101: continuation state not found for %s", display[0])
		}
		if source.Running() {
			return "", nil, nil, 0, fmt.Errorf("analysis is running")
		}
		if len(display) > 1 {
			step, err := strconv.Atoi(display[1])
			if len(display) != 2 || err != nil || step <= 0 || step != source.ContinuationStep() {
				return "", nil, nil, 0, fmt.Errorf(":arclm101: display the latest step %s@%d to continue; older steps have no saved material state", display[0], source.ContinuationStep())
			}
		}
		if source != af {
			af.RestoreState(source.SaveState())
		}
		offset = af.ContinuationStep()
	}
	if output == "" && offset > 0 {
		output = frame.ResultFileName[per]
	}
	if output == "" {
		output = Ce(frame.Path, ".otp")
	}
	cond.SetOutput([]string{output})
	return per, af, cond, offset, nil
}
