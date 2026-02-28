package cri

import (
	"context"
	"fmt"

	"log/slog"

	"github.com/go-logr/logr"

	grpclogging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"google.golang.org/grpc"
	grpcinsecure "google.golang.org/grpc/credentials/insecure"

	criapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"github.com/urfave/cli/v3"
)

const (
	DefaultCRIOAddress       = "unix:///run/crio/crio.sock"
	DefaultContainerdAddress = "unix:///run/containerd/containerd.sock"
)

var (
	DefaultCRIAddress = DefaultCRIOAddress

	DefaultContainerImage = "docker.io/library/ubuntu:latest"
)

type Client interface {
	criapi.RuntimeServiceClient
	criapi.ImageServiceClient

	Conn() *grpc.ClientConn
}

type client struct {
	criapi.RuntimeServiceClient
	criapi.ImageServiceClient

	conn *grpc.ClientConn
}

func (c *client) Conn() *grpc.ClientConn {
	return c.conn
}

func NewClient(address string, logger logr.Logger) (Client, error) {
	if address == "" {
		address = DefaultCRIAddress
	}
	switch address {
	case "crio", "cri-o":
		address = DefaultCRIOAddress
	case "containerd", "c-d":
		address = DefaultContainerdAddress
	}

	glogLogger := grpcLogger(logger.WithName("grpc"))
	glogOpts := []grpclogging.Option{
		grpclogging.WithLogOnEvents(grpclogging.StartCall, grpclogging.FinishCall),
		grpclogging.WithLevels(grpclogging.DefaultClientCodeToLevel),
	}

	conn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(grpcinsecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(
			grpclogging.UnaryClientInterceptor(glogLogger, glogOpts...),
		),
		grpc.WithChainStreamInterceptor(
			grpclogging.StreamClientInterceptor(glogLogger, glogOpts...),
		),
	)
	if err != nil {
		logger.Error(err, "Cannot create gRPC client")
		return nil, err
	}

	return &client{
		criapi.NewRuntimeServiceClient(conn),
		criapi.NewImageServiceClient(conn),
		conn,
	}, nil
}

func grpcLogger(logger logr.Logger) grpclogging.Logger {
	slogLogger := slog.New(logr.ToSlogHandler(logger))
	return grpclogging.LoggerFunc(func(ctx context.Context, level grpclogging.Level, msg string, fields ...any) {
		slogLogger.Log(ctx, slog.Level(level), msg, fields...)
	})
}

func NewCommand() *cli.Command {
	var criClient Client
	var logger logr.Logger

	return &cli.Command{
		Name: "cri",
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			var err error
			logger = logr.FromContextOrDiscard(ctx)
			criClient, err = NewClient("", logger)
			if err != nil {
				return nil, err
			}
			return ctx, nil
		},
		Commands: []*cli.Command{
			{
				Name: "version",
				Action: func(ctx context.Context, c *cli.Command) error {
					reqeust := &criapi.VersionRequest{}
					ver, err := criClient.Version(ctx, reqeust)
					if err != nil {
						return err
					}
					logger.Info("Version", "version", ver)
					return nil
				},
			},
			{
				Name: "runtime-config",
				Action: func(ctx context.Context, c *cli.Command) error {
					req := &criapi.RuntimeConfigRequest{}
					rsp, err := criClient.RuntimeConfig(ctx, req)
					if err != nil {
						return err
					}
					logger.Info("Runtime config", "response", rsp)
					return nil
				},
			},
			{
				Name: "images",
				Action: func(ctx context.Context, c *cli.Command) error {
					request := &criapi.ListImagesRequest{}
					logger.Info("List images", "request", request)
					images, err := criClient.ListImages(ctx, request)
					if err != nil {
						return err
					}
					for _, image := range images.Images {
						logger.Info("Image", "image", image)
					}
					return nil
				},
			},
			{
				Name: "runp",
				Action: func(ctx context.Context, c *cli.Command) error {
					res, err := criClient.RunPodSandbox(ctx, &criapi.RunPodSandboxRequest{
						Config: &criapi.PodSandboxConfig{
							Metadata: &criapi.PodSandboxMetadata{
								Name:      "test-pod",
								Uid:       "test-id",
								Namespace: "test-ns",
							},
							Labels: map[string]string{
								"pod-name":      "test-pod",
								"pod-namespace": "test-ns",
							},
							Linux: &criapi.LinuxPodSandboxConfig{},
						},
					})
					if err != nil {
						return err
					}
					logger.Info("Pod", "id", res.PodSandboxId)
					return nil
				},
			},
			{
				Name: "rmp",
				Action: func(ctx context.Context, c *cli.Command) error {
					pods, err := criClient.ListPodSandbox(ctx, &criapi.ListPodSandboxRequest{
						Filter: &criapi.PodSandboxFilter{
							LabelSelector: map[string]string{
								"pod-name":      "test-pod",
								"pod-namespace": "test-ns",
							},
						},
					})
					if err != nil {
						return err
					}
					if len(pods.Items) != 1 {
						return fmt.Errorf("pod not found")
					}
					podID := pods.Items[0].Id
					logger.Info("Pod", "id", podID)
					_, err = criClient.RemovePodSandbox(ctx, &criapi.RemovePodSandboxRequest{
						PodSandboxId: podID,
					})
					return err
				},
			},
		},
	}
}
