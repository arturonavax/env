package osconfig

import (
	"context"
	"os/exec"
	"time"
)

func defaultExec(ctx context.Context, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).Output()
}
