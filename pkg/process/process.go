package process

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
)

type ProcessManager struct {
	PreExec Process
	Main    Process
}

type Process struct {
	Command []string
}

func (pm *ProcessManager) Run(ctx context.Context) error {
	// preExec
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
		slog.Debug("context canceled")
	}()

	preExec := exec.CommandContext(ctx, pm.PreExec.Command[0], pm.PreExec.Command[1:]...)
	preExec.Env = os.Environ()
	preExec.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err := preExec.Start()
	if err != nil {
		return fmt.Errorf("failed to start pre-exec command: %w", err)
	}
	slog.Debug("pre-exec command started", slog.Int("pid", preExec.Process.Pid))

	// main
	mainCmd := exec.CommandContext(ctx, pm.Main.Command[0], pm.Main.Command[1:]...)
	mainCmd.Stdin = os.Stdin
	mainCmd.Stdout = os.Stdout
	mainCmd.Stderr = os.Stderr
	mainCmd.Env = os.Environ()
	mainCmdDone := make(chan error, 1)

	err = mainCmd.Start()
	if err != nil {
		return fmt.Errorf("failed to run main command: %w", err)
	}
	slog.Debug("main command started", slog.Int("pid", preExec.Process.Pid))

	go func() {
		mainCmdDone <- mainCmd.Wait()
	}()

	select {
	case err := <-mainCmdDone:
		if err != nil {
			return fmt.Errorf("failed to wait main command: %w", err)
		}
		cancel()
		slog.Debug("main command finished")
	case <-ctx.Done():
		slog.Debug("context canceled")
		return nil
	}
	preExecProcess := preExec.Process
	if preExecProcess == nil {
		slog.Debug("pre-exec is already finished")
		return nil
	}
	slog.Debug("pre-exec process terminating...", slog.Int("pid", preExecProcess.Pid))
	err = preExecProcess.Signal(syscall.SIGTERM)
	slog.Debug("send SIGTERM to pre-exec command", slog.Int("pid", preExecProcess.Pid))

	if err != nil {
		slog.Error("failed to send SIGTERM pre-exec command", slog.Int("pid", preExecProcess.Pid), slog.String("error", err.Error()))
		return fmt.Errorf("failed to send SIGTERM pre-exec command: %w", err)
	}

	return nil
}
