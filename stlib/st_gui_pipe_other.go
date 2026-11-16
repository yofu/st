//go:build !windows

package st

import (
	"context"
	"fmt"
)

func RunGUIPipe(_ context.Context, _ string, _ func(GUIRequest) GUIResponse, ready chan<- error) {
	ready <- fmt.Errorf("ST GUI named pipes require Windows")
}
func ExchangeGUI(context.Context, string, GUIRequest) (GUIResponse, error) {
	return GUIResponse{}, fmt.Errorf("ST GUI named pipes require Windows")
}
