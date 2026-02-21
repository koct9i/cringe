package run

import (
	"context"
	"math/rand"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Context("Dummy", Label("dummy"), func() {
	logger := GinkgoLogr

	It("Always succeed", Label("ok"), func() {
		logger.Info("Hello")
		Expect(true).NotTo(BeFalse())
		Expect(false).NotTo(BeTrue())
	})

	It("Always fails", Label("fail"), func() {
		logger.Info("Hello")
		Fail("as expected")
	})

	It("Fails randomly", Label("random"), func() {
		if rand.Intn(6) == 0 {
			Fail("Bang!")
		}
	})

	It("Fails randomly but deterministic depending on seed", Label("random", "deterministic"), func() {
		if NewSeededRand().Intn(6) == 0 {
			Fail("Bang!")
		}
	})

	It("Always skipped", Label("skip"), func() {
		Skip("As usual")
	})

	It("Always stuck", Label("timeout"), func(ctx context.Context) {
		logger.Info("Hello")
		<-ctx.Done()
	})
})
