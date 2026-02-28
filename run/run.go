package run

import (
	"context"
	hashfnv "hash/fnv"
	"log/slog"
	"math/rand"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/urfave/cli/v3"

	"github.com/onsi/ginkgo/v2"

	"github.com/onsi/gomega"
)

type fakeT struct{}

func (fakeT) Fail() {
}

func HashCurrentSpec() int64 {
	h := hashfnv.New64a()
	h.Write([]byte(strings.Join(ginkgo.CurrentSpecReport().ContainerHierarchyTexts, "\n")))
	return int64(h.Sum64()) //nolint:gosec
}

func NewSeededRand() *rand.Rand {
	return rand.New(rand.NewSource(ginkgo.GinkgoRandomSeed() ^ HashCurrentSpec()))
}

func NewCommand() *cli.Command {
	suiteConfig, reporterConfig := ginkgo.GinkgoConfiguration()
	return &cli.Command{
		Name: "run",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "dry-run",
				Aliases:     []string{"n"},
				Destination: &suiteConfig.DryRun,
			},
			&cli.BoolWithInverseFlag{
				Name:        "fail-fast",
				Destination: &suiteConfig.FailFast,
			},
			&cli.IntFlag{
				Name:        "flake-attempts",
				Destination: &suiteConfig.FlakeAttempts,
			},
			&cli.IntFlag{
				Name:        "repeat",
				Aliases:     []string{"r"},
				Destination: &suiteConfig.MustPassRepeatedly,
			},
			&cli.DurationFlag{
				Name:        "timeout",
				Value:       time.Second * 60,
				Aliases:     []string{"t"},
				Destination: &suiteConfig.Timeout,
			},
			&cli.DurationFlag{
				Name:        "poll-progress-after",
				Value:       time.Second * 60,
				Destination: &suiteConfig.PollProgressAfter,
			},
			&cli.DurationFlag{
				Name:        "poll-progress-interval",
				Value:       time.Second * 30,
				Destination: &suiteConfig.PollProgressInterval,
			},
			&cli.Int64Flag{
				Name: "random-seed",
				Action: func(ctx context.Context, c *cli.Command, i int64) error {
					suiteConfig.RandomSeed = i
					return nil
				},
			},
			&cli.BoolWithInverseFlag{
				Name:        "shuffle",
				Aliases:     []string{"s"},
				Destination: &suiteConfig.RandomizeAllSpecs,
			},
			&cli.StringFlag{
				Name:        "label",
				Aliases:     []string{"l"},
				Destination: &suiteConfig.LabelFilter,
			},
			&cli.BoolFlag{
				Name:        "verbose",
				Aliases:     []string{"v"},
				Destination: &reporterConfig.Verbose,
			},
			&cli.BoolFlag{
				Name:        "very-verbose",
				Aliases:     []string{"vv"},
				Destination: &reporterConfig.VeryVerbose,
			},
			&cli.BoolFlag{
				Name:        "quiet",
				Aliases:     []string{"q"},
				Destination: &reporterConfig.Succinct,
			},
			&cli.BoolWithInverseFlag{
				Name:        "silence-skips",
				Value:       true,
				Destination: &reporterConfig.SilenceSkips,
			},
			&cli.BoolWithInverseFlag{
				Name:        "github-output",
				Sources:     cli.EnvVars("GITHUB_ACTIONS"),
				Destination: &reporterConfig.GithubOutput,
			},
			&cli.BoolWithInverseFlag{
				Name:  "color",
				Value: true,
				Action: func(ctx context.Context, c *cli.Command, b bool) error {
					reporterConfig.NoColor = !b
					return nil
				},
			},
			&cli.BoolWithInverseFlag{
				Name:        "force-newlines",
				Value:       true,
				Destination: &reporterConfig.ForceNewlines,
			},
		},
		Arguments: []cli.Argument{
			&cli.StringArgs{
				Name:        "focus",
				Max:         -1,
				Destination: &suiteConfig.FocusStrings,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			ginkgo.GinkgoLogr = logr.FromSlogHandler(slog.NewJSONHandler(ginkgo.GinkgoWriter, &slog.HandlerOptions{
				Level: slog.Level(-logr.FromContextOrDiscard(ctx).GetV()),
			}))
			gomega.RegisterFailHandler(ginkgo.Fail)
			if !ginkgo.RunSpecs(fakeT{}, "", suiteConfig, reporterConfig) {
				return cli.Exit("", 1)
			}
			return nil
		},
	}
}
