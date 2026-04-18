package cri

import (
	"context"
	"io"
	"os"

	"github.com/go-logr/logr"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"

	criapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"github.com/Masterminds/semver/v3"
)

func BeSemVer() gomegatypes.GomegaMatcher {
	return WithTransform(semver.NewVersion, Not(BeNil()))
}

var _ = Context("CRI", Label("cri"), func() {
	var logger logr.Logger
	var client Client

	BeforeEach(func(ctx context.Context) {
		var err error
		logger = GinkgoLogr
		client, err = NewClient("", logger)
		Expect(err).To(Succeed())
	})

	It("version", Label("version"), func(ctx context.Context) {
		version, err := client.Version(ctx, &criapi.VersionRequest{})
		Expect(err).To(Succeed())

		logger.Info("Version", "version", version)
		Expect(version.RuntimeName).ToNot(BeEmpty(), "runtime name")
		Expect(version.Version).To(BeSemVer(), "version")
		Expect(version.RuntimeVersion).To(BeSemVer(), "runtime version")
		Expect(version.RuntimeApiVersion).To(BeSemVer(), "runtime api version")
	})

	It("pull", Label("image"), func(ctx context.Context) {
		imageSpec := &criapi.ImageSpec{
			Image: DefaultContainerImage,
			// UserSpecifiedImage: DefaultContainerImage,
		}

		By("getting image status", func() {
			request := &criapi.ImageStatusRequest{
				Image: imageSpec,
			}
			logger.Info("Image status", "request", request)
			result, err := client.ImageStatus(ctx, request)
			Expect(err).ToNot(HaveOccurred())
			logger.Info("Image status", "result", result)
		})

		By("pulling image", func() {
			request := &criapi.PullImageRequest{
				Image: imageSpec,
			}
			logger.Info("Pull image", "request", request)
			result, err := client.PullImage(ctx, request)
			Expect(err).ToNot(HaveOccurred())
			logger.Info("Pull image", "result", result)
		})

		By("getting image status", func() {
			request := &criapi.ImageStatusRequest{
				Image: imageSpec,
			}
			logger.Info("Image status", "request", request)
			result, err := client.ImageStatus(ctx, request)
			Expect(err).ToNot(HaveOccurred())
			logger.Info("Image status", "result", result)
		})
	})

	It("run itself", Label("run"), func(ctx context.Context) {
		logDirectory := GinkgoT().TempDir()
		Expect(os.Chmod(logDirectory, 0770|os.ModeSetgid)).To(Succeed())

		sbConfig := criapi.PodSandboxConfig{
			Metadata: &criapi.PodSandboxMetadata{
				Namespace: "test",
				Name:      "test",
				Uid:       "test",
			},
			LogDirectory: logDirectory,
		}
		ctConfig := criapi.ContainerConfig{
			Metadata: &criapi.ContainerMetadata{
				Name: "test",
			},
			Image: &criapi.ImageSpec{
				Image:              DefaultContainerImage,
				UserSpecifiedImage: DefaultContainerImage,
			},
			Command: []string{"date"},
		}
		var sbID, ctID string

		By("running pod sandbox", func() {
			result, err := client.RunPodSandbox(ctx, &criapi.RunPodSandboxRequest{Config: &sbConfig})
			Expect(err).ToNot(HaveOccurred())
			sbID = result.PodSandboxId
			logger.Info("Pod sandbox", "id", sbID)
		})
		DeferCleanup(func(ctx context.Context) error {
			_, err := client.RemovePodSandbox(ctx, &criapi.RemovePodSandboxRequest{PodSandboxId: sbID})
			return err
		})

		By("creating container", func() {
			result, err := client.CreateContainer(ctx, &criapi.CreateContainerRequest{
				PodSandboxId:  sbID,
				SandboxConfig: &sbConfig,
				Config:        &ctConfig,
			})
			Expect(err).ToNot(HaveOccurred())
			ctID = result.ContainerId
			logger.Info("Container", "id", ctID)
		})
		DeferCleanup(func(ctx context.Context) error {
			_, err := client.RemoveContainer(ctx, &criapi.RemoveContainerRequest{ContainerId: ctID})
			return err
		})

		events, err := client.GetContainerEvents(ctx, &criapi.GetEventsRequest{})
		Expect(err).ToNot(HaveOccurred())

		By("starting container", func() {
			_, err := client.StartContainer(ctx, &criapi.StartContainerRequest{ContainerId: ctID})
			Expect(err).ToNot(HaveOccurred())
		})

		var logFile io.ReadSeekCloser
		readLog := func() {
			io.Copy(GinkgoWriter, io.LimitReader(logFile, 1<<20))
		}

		By("getting container status", func() {
			result, err := client.ContainerStatus(ctx, &criapi.ContainerStatusRequest{ContainerId: ctID})
			Expect(err).ToNot(HaveOccurred())
			status := result.Status
			logger.Info("Container status", "status", status)
			logger.Info("Container state",
				"state", status.State.String(),
				"reason", status.Reason,
				"message", status.Message,
				"exitCode", status.ExitCode,
				"logPath", status.LogPath,
			)
			logFile, err = os.Open(status.LogPath)
			Expect(err).ToNot(HaveOccurred())
			readLog()
		})

		Eventually(ctx, events.Recv).Should(Satisfy(func(event *criapi.ContainerEventResponse) bool {
			logger.Info("Container event", "type", event.ContainerEventType.String(), "event", event)
			return event.ContainerId == ctID && event.ContainerEventType == criapi.ContainerEventType_CONTAINER_STOPPED_EVENT
		}))

		By("getting container status", func() {
			result, err := client.ContainerStatus(ctx, &criapi.ContainerStatusRequest{ContainerId: ctID})
			Expect(err).ToNot(HaveOccurred())
			status := result.Status
			logger.Info("Container state", "state", status.State.String(), "reason", status.Reason, "message", status.Message, "exitCode", status.ExitCode)
		})
	})
})
