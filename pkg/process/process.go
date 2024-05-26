package process

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type ProcessManager struct {
	PreExec PreExec
	Main    Process
}

type Process struct {
	Command []string
}

type PreExec struct {
	Process
	HealthCheck *HealthCheck
}

type HealthCheckType string

const (
	HealthCheckTypeHTTP HealthCheckType = "http"
	HealthCheckTypeTCP  HealthCheckType = "tcp"
)

type HealthCheck struct {
	Type     HealthCheckType
	Port     int
	Path     string
	Retries  int
	Interval int
	Timeout  int
}

func waitCheckHealth(ctx context.Context, hc *HealthCheck) error {
	for i := 0; i < hc.Retries; i++ {
		switch hc.Type {
		case HealthCheckTypeHTTP:
			err := checkHealthHTTP(ctx, hc)
			if err != nil {
				slog.Warn("failed to check health", slog.String("type", string(hc.Type)), slog.String("path", hc.Path), slog.Int("retries", i), slog.String("error", err.Error()))
			} else {
				return nil
			}
		case HealthCheckTypeTCP:
			err := checkHealthTCP(ctx, hc)
			if err != nil {
				slog.Warn("failed to check health", slog.String("type", string(hc.Type)), slog.String("path", hc.Path), slog.Int("retries", i), slog.String("error", err.Error()))
			} else {
				return nil
			}
		default:
			return fmt.Errorf("unknown healthcheck type: %s", hc.Type)
		}
		time.Sleep(time.Duration(hc.Interval) * time.Second)
	}
	return fmt.Errorf("retry count exceeded")
}

func checkHealthHTTP(ctx context.Context, hc *HealthCheck) error {
	client := http.Client{}
	url := fmt.Sprintf("http://localhost:%d%s", hc.Port, hc.Path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to request: %w", err)
	}
	if res.StatusCode >= 300 {
		return fmt.Errorf("status code is not success: %d", res.StatusCode)
	}
	return nil
}

func checkHealthTCP(_ctx context.Context, hc *HealthCheck) error {
	// unimplemented
	return nil
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

	if pm.PreExec.HealthCheck != nil {
		err = waitCheckHealth(ctx, pm.PreExec.HealthCheck)
		if err != nil {
			return fmt.Errorf("failed to wait healthcheck: %w", err)
		}
	}

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
