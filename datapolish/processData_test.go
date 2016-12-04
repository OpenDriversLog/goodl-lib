package datapolish_test

import (
	"database/sql"
	"math"

	"github.com/kellydunn/golang-geo"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Compufreak345/dbg"
	"github.com/OpenDriversLog/goodl-lib/datapolish"
	"github.com/OpenDriversLog/goodl-lib/dbMan"
	. "github.com/OpenDriversLog/goodl-lib/models/SQLite"
	"github.com/OpenDriversLog/goodl-lib/translate"
)

const pdtTag = "g-lib/polishData_test.go"

/** What do we do here

everything except trackRecords from device 4 "copy of Spreewaldtour" gets deleted. this is to test the complete "ProcessGPSData" function.
Device 2 (PrototestSpreewald) is used for testing `FindKeyPoints` and `createFilteredTrackPoints` (track 240)


*/
var _ = Describe("ProcessData", func() {

	var (
		dbPath string
		dbCon  *sql.DB
	)

	BeforeEach(func() {
		dbPath = "/go/src/github.com/OpenDriversLog/goodl-lib/test_integration/tests.db"
		dbCon, _ = dbMan.GetLocationDb(dbPath,-1)
	})

	AfterEach(func() {
		defer dbCon.Close()
	})

	Describe("ProcessGPSData", func() {

		BeforeEach(func() {
			defer GinkgoRecover()
		})
		T:= &translate.Translater{}

		PContext("device=3", func() {
			It("trying to import device 3 repeatedly", func() {
				defer GinkgoRecover()
				err := datapolish.ProcessGPSData(0, 0, 3, false,1,nil,T, dbCon)
				dbg.I(pdtTag, "reimporting tracks for Device 3 ...", err)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("copy of Spreewaldtour Device == 4", func() {
			// TODO: this one most likely broke when we modified ReProcessThingy
			It("checking Spreewaldtour, 07:33 to 13:11 March 11", func() {
				var err error
				defer GinkgoRecover()

				// By("prepare by removing older DB entries")

				_, err = dbCon.Exec("DELETE FROM KeyPoints WHERE deviceId=4")
				Expect(err).ToNot(HaveOccurred())
				_, err = dbCon.Exec("DELETE FROM trackPoints WHERE (trackId IN (SELECT _trackId FROM tracks WHERE deviceId=4))")
				Expect(err).ToNot(HaveOccurred())
				_, err = dbCon.Exec("DELETE FROM tracks WHERE deviceId=4")
				Expect(err).ToNot(HaveOccurred())

				By("processing data...")
				err = datapolish.ProcessGPSData(0, math.MaxInt64, 4, true,1,nil,T, dbCon)
				Expect(err).ToNot(HaveOccurred())

				var countTrack sql.NullInt64
				var countKeyPoints sql.NullInt64
				By("3 tracks")
				err = dbCon.QueryRow("SELECT COUNT(*) FROM `tracks` WHERE deviceId=?", 4).Scan(&countTrack)
				Expect(countTrack.Int64).To(Equal(int64(4)))

				By("4 KeyPoints")
				err = dbCon.QueryRow("SELECT COUNT(*) FROM `keyPoints` WHERE deviceId=?", 4).Scan(&countKeyPoints)
				if err != nil {
					dbg.E(pdtTag, "Failed to get counts...", err)
				}

				Expect(countKeyPoints.Int64).To(Equal(int64(5)))
			})
		})

	})

	Describe("FindKeyPoints on dev=2", func() {
		Context("spreewaldTour (device == 2)", func() {
			var (
				// startTime    int64
				// endTime      int64
				err          error
				trackrecords []Location
			)

			BeforeEach(func() {
				defer GinkgoRecover()
				trackrecords, err = datapolish.GetTrackRecordsForDevice(0, math.MaxInt64, 2, dbCon)
			})

			It("should find timeRange & trackRecords results", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(err).ToNot(HaveOccurred())
				Expect(trackrecords).ToNot(Equal(nil))
				Expect(trackrecords).Should(HaveLen(21214))
			})

			It("should should give result", func() {
				defer GinkgoRecover()
				result, err := datapolish.FindKeyPoints(dbCon, trackrecords, nil,1)

				By("not empty")

				Expect(err).ToNot(HaveOccurred())
				Expect(result).ToNot(Equal(nil))

				By("2 or more")
				dbg.I(pdtTag, "found %d keypoints...", len(result))
				Expect(len(result)).Should(BeNumerically(">=", 2))

				By("all start/endTime > 0")
				for _, kp := range result {
					Expect(kp.StartTime.Int64).Should(BeNumerically(">", 0))
					Expect(kp.EndTime.Int64).Should(BeNumerically(">", 0))
				}

				By("should NOT have shorter timespan than config, except first & last one", func() {

					for idx, kp := range result {
						if idx > 0 && idx < len(result)-1 {
							dbg.W(pdtTag, "kp idx: %d, id: %d...", idx, kp.KeyPointId.Int64, kp.StartTime.Int64, kp.EndTime.Int64)

							Expect(kp.EndTime.Int64 - kp.StartTime.Int64).Should(BeNumerically(">=", datapolish.GetDefaultLocationConfig().MinMoveTime))
						}
					}
				})
			})

			It("should have specific start KeyPoint", func() {
				defer GinkgoRecover()
				result, err := datapolish.FindKeyPoints(dbCon, trackrecords, nil,1)
				Expect(err).ToNot(HaveOccurred())

				p := geo.NewPoint((result)[0].Latitude.Float64, (result)[0].Longitude.Float64)
				expectedPoint := geo.NewPoint(50.920913, 13.33474)
				Expect(p.GreatCircleDistance(expectedPoint)).Should(BeNumerically("<=", datapolish.GetDefaultLocationConfig().MinMoveDist))
				// Expect((result)[0].EndTime.Int64).Should(Equal(int64(1426059212000)))
			})

			It("should have specific end KeyPoint", func() {
				defer GinkgoRecover()
				result, err := datapolish.FindKeyPoints(dbCon, trackrecords, nil,1)
				lkp := (result)[len(result)-1]
				Expect(err).ToNot(HaveOccurred())
				p := geo.NewPoint(lkp.Latitude.Float64, lkp.Longitude.Float64)
				expectedPoint := geo.NewPoint(50.919455, 13.341798)
				Expect(p.GreatCircleDistance(expectedPoint)).Should(BeNumerically("<=", datapolish.GetDefaultLocationConfig().MinMoveDist))
				Expect(lkp.EndTime.Int64).Should(Equal(int64(1426079437000)))
			})

		}) // context: spreewaldtour (dev 2)

		PContext("partial upload (500 points from paul, device=6)", func() {
			It("should find at least 2 keypoints", func() {
				defer GinkgoRecover()
				trackrecords, err := datapolish.GetTrackRecordsForDevice(0, math.MaxInt64, 6, dbCon)
				kps, err := datapolish.FindKeyPoints(dbCon, trackrecords, nil,1)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(kps)).To(BeNumerically(">=", 2), "#KPs found")
			})
		}) // context: 500 points from paul, device=6

		Context("FS@home march 2-4 (1425252600000 () to 1425439800000 ()", func() {
			It("should find 2 keypoints, which are similar in position", func() {
				defer GinkgoRecover()
				trackrecords, err := datapolish.GetTrackRecordsForDevice(1425252600000, 1425439800000, 3, dbCon)
				kps, err := datapolish.FindKeyPoints(dbCon, trackrecords, nil,1)
				for idx, kp := range kps {
					dur := (kp.EndTime.Int64 - kp.StartTime.Int64)
					dbg.W(pdtTag, "kp idx: %d, id: %d...dur: %d", idx, kp.KeyPointId.Int64, dur)
				}

				// 7: 3x at home + 4x @kita
				Expect(len(kps)).To(BeNumerically("<=", 11), "#KPs found")
				Expect(err).NotTo(HaveOccurred())
			})
		}) // context: FS@home march 10-12

	}) // Describe FindKeyPoints

	Describe("CreateFilteredTrackPoints", func() {

		BeforeEach(func() {
			defer GinkgoRecover()
		})

		// Context("trackId=1", func() {
		// 	It("trying to import tracks repeatedly", func() {
		// 		defer GinkgoRecover()
		// 		_, _ = dbCon.Exec("DELETE FROM `trackPoints` WHERE trackId=1")
		// 		_, err := datapolish.CreateFilteredTrackPoints(1, nil, dbCon)
		// 		dbg.I(pdtTag, "reimporting tracks for Track 1 ...", err)
		// 		Expect(err).To(HaveOccurred())
		// 	})
		// })

		Context("trackId=240 (1st track of ProtoTestSpreewald)", func() {
			It("inserting new trackPoints", func() {
				defer GinkgoRecover()
				By("prepare by removing older DB entries")
				_, err := dbCon.Exec("DELETE FROM trackPoints WHERE trackId=240")
				Expect(err).NotTo(HaveOccurred())

				By("CreateFilteredTrackPoints")
				_, err = datapolish.CreateFilteredTrackPoints(240, nil, dbCon)
				Expect(err).NotTo(HaveOccurred())
				var countTrack sql.NullInt64
				var countKeyPoints sql.NullInt64
				err = dbCon.QueryRow("SELECT COUNT(*) FROM `trackPoints` WHERE deviceId=?", 2).Scan(&countTrack)
				err = dbCon.QueryRow("SELECT COUNT(*) FROM `keyPoints` WHERE deviceId=?", 2).Scan(&countKeyPoints)
				if err != nil {
					dbg.E(pdtTag, "Failed to get counts...", err)
				}

				// Expect(countTrack.Int64).To(Equal(int64(3)))
				// Expect(countKeyPoints.Int64).To(Equal(int64(4 + 2)))
			})
		})

	}) // Describe CreateFilteredTrackPoints

})
