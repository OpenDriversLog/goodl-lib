package datapolish_test

import (
	"database/sql"
	"math"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Compufreak345/dbg"
	"github.com/OpenDriversLog/goodl-lib/datapolish"
)

const gdtTag = "g-lib/getData_test.go"

var _ = Describe("GetData", func() {

	var (
		dbPath string
		db     *sql.DB
	)

	BeforeEach(func() {
		dbPath = "/go/src/github.com/OpenDriversLog/goodl-lib/test_integration/tests.db"
		db, _ = datapolish.OpenDbCon(dbPath)
	})

	AfterEach(func() {
		defer db.Close()
	})

	Describe("GetDevices", func() {
		Context("valid DBpath", func() {
			It("should give result", func() {
				defer GinkgoRecover()
				result, err := datapolish.GetDeviceStrings(db)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).ToNot(Equal(nil))
				Expect(result).Should(HaveKeyWithValue(4, "copy of Spreewaldtour"))

				Expect(result).Should(HaveLen(5))
			})
		})

	})

	// TODO: same problem as GetTrackIdsInTimeRange...
	Describe("GetDeviceTimeRange", func() {
		Context("checking  Spreewaldtour (device 2), 07:33 to 13:11 March 11", func() {
			It("should give result", func() {
				defer GinkgoRecover()
				min, max, err := datapolish.GetDeviceTimeRange(2, db)
				dbg.D(gdtTag, "getdevicetimerange result", min, max)
				Expect(err).ToNot(HaveOccurred())
				Expect(min).Should(BeNumerically(">", int64(0)))
				Expect(max).Should(BeNumerically(">", int64(0)))
				Expect(max).Should(BeNumerically(">", min))

				Expect(min).Should(BeNumerically("<=", int64(1426059212000)))
				Expect(max).Should(BeNumerically(">=", int64(1426079437000)))
			})
		})

	})

	Describe("GetTrackRecordsForDevice", func() {
		Context("valid DBpath", func() {
			It("checking Spreewaldtour (device 2), 07:33 to 13:11 March 11", func() {
				defer GinkgoRecover()
				result, err := datapolish.GetTrackRecordsForDevice(0, math.MaxInt64, 2, db)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).ToNot(Equal(nil))
				Expect(result).Should(HaveLen(21214))
			})
		})

	})

	Describe("GetTrackRecords", func() {
		Context("valid DBpath", func() {
			It("should should give result", func() {
				defer GinkgoRecover()
				result, err := datapolish.GetTrackRecords(0, math.MaxInt64, db)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).ToNot(Equal(nil))

				dbg.D(pdtTag, "lengh of trackRecords", len(result))

			})
			It("should should have same length as getDevices result", func() {
				defer GinkgoRecover()
				result, err := datapolish.GetTrackRecords(0, math.MaxInt64, db)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).ToNot(Equal(nil))

				devices, err := datapolish.GetDeviceStrings(db)
				trlen := len(result)
				devlen := len(devices)

				Expect(trlen).Should(Equal(devlen))
			})
		})
	})

})
