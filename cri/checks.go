package cri

import (
	"context"

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
	logger := GinkgoLogr
	var client Client

	BeforeEach(func(ctx context.Context) {
		var err error
		client, err = NewClient("", logger)
		Expect(err).ToNot(HaveOccurred())
	})

	It("version", Label("version"), func(ctx context.Context) {
		version, err := client.Version(ctx, &criapi.VersionRequest{})
		Expect(err).ToNot(HaveOccurred())
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
})
