//go:build !linux

package network

import (
	"context"
	"fmt"
	"runtime"
)

type RouteSwitcher struct{}

func (RouteSwitcher) CurrentDefaultInterface(context.Context) (string, error) {
	return "", fmt.Errorf("default route query is not supported on %s", runtime.GOOS)
}

func (RouteSwitcher) SwitchDefaultInterface(context.Context, string) error {
	return fmt.Errorf("route switching is not supported on %s", runtime.GOOS)
}

func (RouteSwitcher) VerifyDefaultInterface(context.Context, string) error {
	return fmt.Errorf("route verification is not supported on %s", runtime.GOOS)
}
