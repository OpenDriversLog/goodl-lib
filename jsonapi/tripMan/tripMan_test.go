package tripMan_test

import (
	"database/sql"
	"encoding/json"
	"math"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Compufreak345/dbg"

	"github.com/OpenDriversLog/goodl-lib/dbMan"
	"github.com/OpenDriversLog/goodl-lib/jsonapi"
	"github.com/OpenDriversLog/goodl-lib/jsonapi/tripMan"
	m "github.com/OpenDriversLog/goodl-lib/jsonapi/tripMan/models"

	"github.com/OpenDriversLog/goodl-lib/translate"
)

const tmtTag = "g-lib/tripMan/tripMan_test.go"

var _ = Describe("TripMan", func() {

	var (
		dbPath string
		dbCon  *sql.DB
	)

	BeforeEach(func() {
		dbPath = "/go/src/github.com/OpenDriversLog/goodl-lib/test_integration/tests.db"
		dbCon, _ = dbMan.GetLocationDb(dbPath,-1)

		// this would not work when tests run parallelized
		// _, _ = dbCon.Exec("DELETE FROM `trackPoints` WHERE deviceId=2")
	})

	T:= &translate.Translater{}
	// TODO: implement correctly
	Describe("getTrip (by Id)", func() {
		It("should have all set", func() {
			defer GinkgoRecover()

			/*	trip, err := tripMan.GetTrip(1, false, dbCon)

				Expect(err).ToNot(HaveOccurred())
				Expect(trip).ToNot(Equal(nil))

				Expect(trip.Id).To(Equal(int64(1)))
				Expect(trip.Id).NotTo(BeNil())

				Expect(trip.Type).To(BeNumerically(">=", 0))
				Expect(trip.Type).To(BeNumerically("<", 3))

				Expect(trip.Title).ToNot(BeNil())*/
		})

	}) // Describe CreateFilteredTrackPoints

	// TODO: CS: implement correctly
	Describe("updateTrip", func() {
		Context("some mocked up thing", func() {
			BeforeEach(func() {
				_, err := dbCon.Exec(`
INSERT INTO Trips (type,title,desc,driverId,contactId) VALUES
(3,"testupdate","businesstrip",1,154),
(2,"testupdate","workway",1,154),
(1,"testupdate","private",1,154),
(3,"testupdate","businesstrip_history",1,154);
`)
				Expect(err).To(BeNil())

			})
			AfterEach(func() {
				_, err := dbCon.Exec(`
		DELETE FROM Tracks_Trips WHERE tripId IN (SELECT _tripId FROM Trips WHERE title LIKE 'testupdate%');
DELETE FROM Trip_History WHERE tripId IN (SELECT _tripId FROM Trips WHERE title LIKE "testupdate%");
DELETE FROM Trips WHERE title LIKE "testupdate%";`)
				Expect(err).To(BeNil())
			})
			It("should update all the fields", func() {
				defer GinkgoRecover()

				var id int64

				err := dbCon.QueryRow(`SELECT _tripId FROM Trips WHERE (title='testupdate' AND desc='businesstrip') LIMIT 1`).Scan(&id)
				Expect(err).ToNot(HaveOccurred())

				Expect(id).To(BeNumerically(">", 0))
				_, err = dbCon.Exec(`
INSERT INTO Tracks_Trips (tripId, trackId) VALUES (?,238),(?,239)`, id, id)

				trip, err := tripMan.GetTrip(id, false, false, false,nil,T,true, dbCon)
				Expect(trip).ToNot(BeNil())

				newTrip, _,_,_, err := tripMan.UpdateTrip(&m.Trip{Id: id, Type: tripMan.PRIVATE, Title: "testupdate1", Description: "business to workway", ContactId: 156, DriverId: 1},true,nil,T, dbCon)

				Expect(err).ToNot(HaveOccurred())
				Expect(newTrip).ToNot(BeNil())

				Expect(newTrip.Id).To(Equal(trip.Id))
				Expect(newTrip.Type).To(Equal(1))
				Expect(newTrip.Title).To(BeEquivalentTo("testupdate1"))
				Expect(newTrip.Description).To(BeEquivalentTo("business to workway"))
				Expect(newTrip.TrackIds).To(BeEquivalentTo("238,239"))
			})

			It("should not accept private > business", func() {
				defer GinkgoRecover()

				var id int64
				err := dbCon.QueryRow(`SELECT _tripId FROM Trips WHERE (title='testupdate' AND desc='private')`).Scan(&id)
				_, err = dbCon.Exec(`
INSERT INTO Tracks_Trips (tripId, trackId) VALUES (?,238)`, id)

				_, err = tripMan.GetTrip(id, false, false, false,nil,T,false, dbCon)

				Expect(err).ToNot(HaveOccurred())

				newTrip, _,_,_, err := tripMan.UpdateTrip(&m.Trip{Id: id, Type: tripMan.BUSINESS, Title: "testupdate2", Description: "private to business", ContactId: 155, DriverId: 1},true,nil,T, dbCon)

				if !dbg.Develop {
					Expect(err).To(HaveOccurred())
					Expect(newTrip.Type).To(Equal(1))
				}
			})

			It("should not accept private > workway", func() {
				defer GinkgoRecover()

				var id int64
				err := dbCon.QueryRow(`SELECT _tripId FROM Trips WHERE (title='testupdate' AND desc='private')`).Scan(&id)
				_, err = dbCon.Exec(`
INSERT INTO Tracks_Trips (tripId, trackId) VALUES (?,238)`, id)

				_, err = tripMan.GetTrip(id, false, false, false,nil,T,true, dbCon)

				newTrip, _,_,_, err := tripMan.UpdateTrip(&m.Trip{Id: id, Type: tripMan.BUSINESS, Title: "testupdate3", Description: "private to workway", ContactId: 154, DriverId: 1},true,nil,T, dbCon)
				if !dbg.Develop {
					Expect(err).To(HaveOccurred())
					Expect(newTrip.Type).To(Equal(1))
				}
			})
		})
	}) // Describe CreateFilteredTrackPoints

	// TODO: KPs are probably not inserted, yet again, after they were deleted for dev=2 in processData_test
	PDescribe("GetTrackIdsInTimeRange", func() {
		Context("get Ids for device 2", func() {
			It("should should give result", func() {
				defer GinkgoRecover()
				result, err := tripMan.GetTrackIdsInTimeRange(0, math.MaxInt64, []interface{}{int64(2)}, dbCon)

				By("without error")
				Expect(err).ToNot(HaveOccurred())
				Expect(result).ToNot(Equal(nil))

				By("having length 2")
				trlen := len(result)

				dbg.D(tmtTag, "got datapolish trackIds:", result)
				Expect(trlen).Should(Equal(2))

				// dbg.D(tmtTag, "lengh of trackRecords", len(result))
			})
		})
	}) // Describe: getTrackIdsInTimeRange

	Describe("jsonapi/GetTrackById", func() {

		Context("get jsonData for a track", func() {
			// TODO: implement
			It("should give result", func() {
				defer GinkgoRecover()
				//_, err := datapolish.CreateFilteredTrackPoints(3, nil, dbCon)
				//Expect(err).NotTo(HaveOccurred())
				result, err := jsonapi.GetTrackById(dbCon, 3)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).ToNot(Equal(nil))
				// Expect(result).Should(HaveKeyWithValue("ProtoTestSpreewald", 2))
				// Expect(result).ShouldNot(HaveLen(3))

				dbg.D(tmtTag, "lengh of trackRecords", len(result))

			})

			It("should have all fields", func() {

				/*
					{"type":"FeatureCollection","properties":{"device":1,"track":300}
					,"features":[
					    {"type":"Feature","properties":{"City":"Freiberg","KeyPointId":300,"MatchingContactids":"","MaxTime":1426052820000,"MinTime":1426039045000,"Postal":"09599","Street":"Schloßplatz","class":"marker","name":"Mar 11 01:57:25-Mar 11 05:47:00"}
					    ,{"type":"Feature","properties":{"City":"Freiberg","KeyPointId":301,"MatchingContactids":"","MaxTime":1426053878000,"MinTime":1426052922000,"Postal":"09599","Street":"Burgstraße","class":"marker","name":"Mar 11 05:48:42-Mar 11 06:04:38"}
					    ,"id":"StartKeyPoint","geometry":{"type":"Point","coordinates":[13.340726,50.92027]}}
					    ,"id":"EndKeyPoint","geometry":{"type":"Point","coordinates":[13.34167,50.919395]}}
					]}
				*/
				defer GinkgoRecover()

				//_, err := datapolish.CreateFilteredTrackPoints(242, nil, dbCon)
				//Expect(err).NotTo(HaveOccurred())
				result, err := jsonapi.GetTrackById(dbCon, 239)
				dbg.WTF(tmtTag, "trackdata for track 239 %s", result)

				var unm map[string]interface{}
				err = json.Unmarshal(result, &unm)

				Expect(err).ToNot(HaveOccurred())

				Expect(unm["id"]).ToNot(Equal(""))
				dbg.WTF("Tests", "unmarshaled track 239", unm)
				Expect(len(unm)).To(BeNumerically(">=", 0))
				Expect(unm["features"].([]interface{})[0].(map[string]interface{})["id"]).To(BeEquivalentTo("StartKeyPoint"))
				Expect(unm["features"].([]interface{})[1].(map[string]interface{})["id"]).To(BeEquivalentTo("EndKeyPoint"))

				unm = unm["features"].([]interface{})[0].(map[string]interface{})["properties"].(map[string]interface{})
				Expect(unm["KeyPointId"]).ToNot(Equal(""))
				Expect(unm["name"]).ToNot(BeNil())

				// err = json.Unmarshal(unm["EndGeoPoint"], egeo)
				// res := make(map[string]map[string]interface{})
				// json.Unmarshal(result, &res)
				// TODO: try conversion, sth. not working correctly here
				// slat, _ := res["StartGeoPoint"]["lat"].(float64)
				// Expect(slat).To(Equal(float64(50.90187)))
				// Expect(unm["EndGeoPoint"]["lng"]).ToNot(Equal(""))
				// Expect(unm["id"].(int)).To(Equal(int64(1)))
				// Expect(unm["StartKeyPointId"].(int)).To(Equal(int64(1)))
				// Expect(unm["EndKeyPointId"].(int)).To(Equal(int64(2)))
			})

		})

	})

	Describe("jsonapi/GetTrackIdsInTimeRange", func() {

		Context("get ids for a track", func() {
			// TODO: implement
			It("should give result", func() {
				defer GinkgoRecover()
				result, err := jsonapi.GetTrackIdsInTimeRange(0, math.MaxInt64, []interface{}{int64(2)}, dbCon)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).ToNot(Equal(nil))
				// Expect(result).Should(HaveKeyWithValue("ProtoTestSpreewald", 2))
				// Expect(result).ShouldNot(HaveLen(2))

				var unm map[string]interface{}
				err = json.Unmarshal(result, &unm)
				dbg.W(tmtTag, "got TrackIds for device=2", unm)

			})
		})
	})

})
