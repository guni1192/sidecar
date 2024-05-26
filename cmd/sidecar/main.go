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
	healthcheck string
	mainCmd     string
	isDebug     bool
)

var rootCmd = &cobra.Command{
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

		preExecArgs := strings.Split(preExec, " ")
		psMgr := process.ProcessManager{
			PreExec: process.Process{Command: preExecArgs},
			Main:    process.Process{Command: args},
		}

		ctx := context.Background()
		return psMgr.Run(ctx)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&preExec, "pre-exec", "", "", "pre-exec command")
	rootCmd.PersistentFlags().BoolVarP(&isDebug, "debug", "", false, "show debug log")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v", err)
		os.Exit(1)
	}
}
