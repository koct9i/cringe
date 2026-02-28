package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"time"

	"log/slog"

	"github.com/go-logr/logr"

	"github.com/urfave/cli/v3"

	"github.com/koct9i/cringe/cri"
	"github.com/koct9i/cringe/run"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var logger logr.Logger
	var verbosity int

	var timeout time.Duration
	var ctxCancel func()

	command := cli.Command{
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:        "verbosity",
				Aliases:     []string{"v"},
				Destination: &verbosity,
			},
			&cli.StringFlag{
				Name:        "cri-address",
				Usage:       "Socket address or service name: crio, cri-o, c-d, containerd",
				Aliases:     []string{"runtime-endpoint", "r"},
				Sources:     cli.EnvVars("CONTAINER_RUNTIME_ENDPOINT"),
				Value:       cri.DefaultCRIAddress,
				Destination: &cri.DefaultCRIAddress,
			},
			&cli.StringFlag{
				Name:        "default-image",
				Value:       cri.DefaultContainerImage,
				Destination: &cri.DefaultContainerImage,
			},
			&cli.DurationFlag{
				Name:        "timeout",
				Aliases:     []string{"t"},
				Destination: &timeout,
			},
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			if timeout > 0 {
				ctx, ctxCancel = context.WithTimeout(ctx, timeout)
			}
			logger = logr.FromSlogHandler(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.Level(-verbosity),
			}))
			ctx = logr.NewContext(ctx, logger)
			return ctx, nil
		},
		After: func(ctx context.Context, c *cli.Command) error {
			if ctxCancel != nil {
				ctxCancel()
			}
			return nil
		},
		Commands: []*cli.Command{
			cri.NewCommand(),
			run.NewCommand(),
		},
	}
	args := append(strings.Split(os.Args[0], "__"), os.Args[1:]...)
	if err := command.Run(ctx, args); err != nil {
		logger.Error(err, "Failed")
	}
}
