package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/guni1192/sidecar/pkg/process"
	"github.com/spf13/cobra"
)

var (
	preExec     string
	mainCmd     string
	isDebug     bool
	healthcheck bool
	timeout     int
	interval    int
	retries     int
	path        string
	port        int

	rootCmd = &cobra.Command{
		Use:   "sidecar",
		Short: "sidecar is onshot multi-process manager",
		RunE: func(cmd *cobra.Command, args []string) error {
			level := slog.LevelInfo
			if isDebug {
				level = slog.LevelDebug
			}

			logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: level,
			}))
			slog.SetDefault(logger)
			slog.Debug("start sidecar", slog.String("pre-exec", preExec), slog.String("main", strings.Join(args, " ")))

			var hc *process.HealthCheck
			if healthcheck {
				hc = &process.HealthCheck{
					Type:     process.HealthCheckTypeHTTP,
					Port:     port,
					Path:     path,
					Retries:  retries,
					Interval: interval,
					Timeout:  timeout,
				}
			}

			preExecArgs := strings.Split(preExec, " ")
			psMgr := process.ProcessManager{
				PreExec: process.PreExec{
					Process: process.Process{
						Command: preExecArgs,
					},
					HealthCheck: hc,
				},
				Main: process.Process{Command: args},
			}

			ctx := context.Background()
			return psMgr.Run(ctx)
		},
	}
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&preExec, "pre-exec", "", "", "pre-exec command")
	rootCmd.PersistentFlags().BoolVarP(&isDebug, "debug", "", false, "show debug log")
	rootCmd.PersistentFlags().BoolVarP(&healthcheck, "healthcheck", "", false, "enable healthcheck")
	rootCmd.PersistentFlags().IntVarP(&timeout, "timeout", "", 10, "timeout seconds")
	rootCmd.PersistentFlags().IntVarP(&interval, "interval", "", 1, "interval seconds")
	rootCmd.PersistentFlags().IntVarP(&retries, "retries", "", 5, "retries")
	rootCmd.PersistentFlags().StringVarP(&path, "path", "", "/", "path")
	rootCmd.PersistentFlags().IntVarP(&port, "port", "", 8000, "port")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v", err)
		os.Exit(1)
	}
}
