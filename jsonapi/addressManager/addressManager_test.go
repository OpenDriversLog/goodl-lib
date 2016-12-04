package addressManager_test

import (
	"database/sql"
	"encoding/json"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/OpenDriversLog/goodl-lib/dbMan"
	. "github.com/OpenDriversLog/goodl-lib/jsonapi/addressManager"
)

var _ = Describe("AddressManager", func() {
	var (
		dbPath string
		dbCon  *sql.DB
	)

	BeforeEach(func() {
		dbPath = "/go/src/github.com/OpenDriversLog/goodl-lib/test_integration/tests.db"
		dbCon, _ = dbMan.GetLocationDb(dbPath,-1)
		GeoZoneSize = 0.05
	})

	AfterEach(func() {
		defer dbCon.Close()
	})
	Describe("CreateReadContact", func() {
		BeforeEach(func() {
			defer GinkgoRecover()
			_, err := dbCon.Exec(`DELETE FROM GeoFenceRegions WHERE outerMinLat IN (0.1330153751649422,0.1333371537,0.1333371538);
			DELETE FROM Rectangles WHERE topLeftLat IN(0.1330153751649422,0.1333371537,0.1333371538);
			DELETE FROM Contacts WHERE title LIKE 'MegaHyperTestContact%';
			DELETE FROM Addresses WHERE street LIKE 'Megahypererfundene Straße%'`)
			Expect(err).To(BeNil())

		})

		It("Should create contact and automagically address and GeoFence!", func() {
			defer GinkgoRecover()
			c := Contact{
				Address: &Address{
					Latitude:    0.13333333337,
					Longitude:   -0.13333333337,
					Street:      "Megahypererfundene Straße",
					Postal:      "01337",
					City:        "Übelste City Oida",
					Additional1: "Additional1",
					Additional2: "Additional2",
					HouseNumber: "13",
					Title:       "AddTitle",
				},
				Title:       "MegaHyperTestContact",
				Description: "Desc",
				Additional:  "Additional",
				TripType:    1,
				Type:        2,
			}

			_, err := CreateContact(&c, dbCon)
			Expect(err).To(BeNil())
			By("Creation should have set Id of Address,GeoZone and GeoRectangle", func() {
				Expect(c.Address.Id).To(BeNumerically(">", 0))
				Expect(c.Address.GeoZones[0].Id).To(BeNumerically(">", 0))
				Expect(c.Address.GeoZones[0].Rectangle.Id).To(BeNumerically(">", 0))
			})
			By("Address table", func() {
				defer GinkgoRecover()
				r := dbCon.QueryRow("SELECT latitude,longitude,street,postal,city,additional1,additional2,houseNumber,title FROM Addresses WHERE street='Megahypererfundene Straße'")
				var lat, ln float64
				var st, pos, cit, add1, add2, hNum, title string
				err := r.Scan(&lat, &ln, &st, &pos, &cit, &add1, &add2, &hNum, &title)
				Expect(err).To(BeNil())
				Expect(lat).To(BeEquivalentTo(0.13333333337))
				Expect(ln).To(BeEquivalentTo(-0.13333333337))
				Expect(st).To(BeEquivalentTo("Megahypererfundene Straße"))
				Expect(pos).To(BeEquivalentTo("01337"))
				Expect(cit).To(BeEquivalentTo("Übelste City Oida"))
				Expect(add1).To(BeEquivalentTo("Additional1"))
				Expect(add2).To(BeEquivalentTo("Additional2"))
				Expect(hNum).To(BeEquivalentTo("13"))
				Expect(title).To(BeEquivalentTo("AddTitle"))
			})

			By("Contact table", func() {
				defer GinkgoRecover()
				r := dbCon.QueryRow("SELECT type,title,description,additional,addressId,tripTypeId FROM Contacts WHERE title='MegaHyperTestContact'")
				var _type, tripTypeId, addrId int64
				var title, desc, add string

				err := r.Scan(&_type, &title, &desc, &add, &addrId, &tripTypeId)
				Expect(err).To(BeNil())
				Expect(_type).To(BeEquivalentTo(2))
				Expect(title).To(BeEquivalentTo("MegaHyperTestContact"))
				Expect(desc).To(BeEquivalentTo("Desc"))
				Expect(add).To(BeEquivalentTo("Additional"))
				Expect(addrId).To(BeNumerically(">", 0))
				Expect(tripTypeId).To(BeEquivalentTo(1))
			})

			By("GeoFence with rectangle at address for contact", func() {
				defer GinkgoRecover()
				r := dbCon.QueryRow(`SELECT outerMinLat,outerMinLon,outerMaxLat,outerMaxLon,
				rectangleId,circleId,topLeftLat,topLeftLon,botRightLat,botRightLon FROM
				Contacts c LEFT JOIN Address_GeoFenceRegion a ON a.addressId = c.addressId
				LEFT JOIN GeoFenceRegions ON geoFenceRegionId=_geoFenceRegionId
				LEFT JOIN Rectangles ON rectangleId=_rectangleId WHERE c.title='MegaHyperTestContact'`)
				var outMinLat, outMinLon, outMaxLat, outMaxLon, topLeftLat, topLeftLon, botRightLat, botRightLon float64
				var rectId int64
				var circleId sql.NullInt64
				err := r.Scan(&outMinLat, &outMinLon, &outMaxLat, &outMaxLon, &rectId, &circleId, &topLeftLat,
					&topLeftLon, &botRightLat, &botRightLon)
				Expect(err).To(BeNil())
				//0.1330153751649422	-0.13365129242981638	0.13365129157095162	-0.33015374301
				Expect(outMinLat).To(BeEquivalentTo(0.1330153751649422))
				Expect(outMinLon).To(BeEquivalentTo(-0.13365129242981638))
				Expect(outMaxLat).To(BeEquivalentTo(0.13365129157095162))
				Expect(outMaxLon).To(BeEquivalentTo(-0.1330153743019036))
				Expect(topLeftLat).To(BeEquivalentTo(0.1330153751649422))
				Expect(topLeftLon).To(BeEquivalentTo(-0.13365129242981638))
				Expect(botRightLat).To(BeEquivalentTo(0.13365129157095162))
				Expect(botRightLon).To(BeEquivalentTo(-0.1330153743019036))
			})
		})

		It("Should be able to read contact with GeoZones & address correctly", func() {
			defer GinkgoRecover()
			_, err := dbCon.Exec(`INSERT INTO "Addresses"(street,postal,city,additional1,additional2,latitude,longitude,HouseNumber,title)
			 VALUES('Megahypererfundene Straße2','01337','Übelste City Oida',
			'Additional1','Additional2',0.13333333337,-0.13333333337,'13','AddTitle');
			INSERT INTO "Rectangles" (topLeftLat,topLeftLon,botRightLat,botRightLon) VALUES(0.1333371537,-0.13365129242981638,0.13365129157,-0.33015374301);

INSERT INTO "GeoFenceRegions" (outerMinLat,outerMinLon,outerMaxLat,outerMaxLon,rectangleId,circleId) VALUES(0.1333371537,-0.13365129242981638,0.13365129157,-0.33015374301,
(SELECT _rectangleId FROM Rectangles WHERE topLeftLat = 0.1333371537 LIMIT 1),
NULL);
INSERT INTO Address_GeoFenceRegion (geoFenceRegionId,addressId) VALUES (
(SELECT _geoFenceRegionId FROM GeoFenceRegions WHERE outerMinLat = 0.1333371537 LIMIT 1),
(SELECT _addressId FROM Addresses where street='Megahypererfundene Straße2' LIMIT 1)
);
INSERT INTO "Contacts"(type,title,description,additional,addressId,tripTypeId)
 VALUES(2,'MegaHyperTestContact2','Desc','Additional',
(SELECT _addressId from Addresses where street='Megahypererfundene Straße2' LIMIT 1),
1);
			`)
			Expect(err).To(BeNil())

			By("Completely with GeoZones", func() {
				defer GinkgoRecover()
				contacts, err := GetContactsWithGeoZones("street='Megahypererfundene Straße2'", dbCon)
				Expect(err).To(BeNil())
				Expect(len(contacts)).To(BeEquivalentTo(1))
				c := contacts[0]

				Expect(c.Address.Id).To(BeNumerically(">", 0))
				Expect(len(c.Address.GeoZones)).To(BeEquivalentTo(1))
				Expect(c.Address.GeoZones[0].Id).To(BeNumerically(">", 0))
				Expect(c.Address.GeoZones[0].Rectangle.Id).To(BeNumerically(">", 0))
				Expect(c.Address.Latitude).To(BeEquivalentTo(0.13333333337))
				Expect(c.Address.Longitude).To(BeEquivalentTo(-0.13333333337))
				Expect(c.Address.Street).To(BeEquivalentTo("Megahypererfundene Straße2"))
				Expect(c.Address.Postal).To(BeEquivalentTo("01337"))
				Expect(c.Address.City).To(BeEquivalentTo("Übelste City Oida"))
				Expect(c.Address.Title).To(BeEquivalentTo("AddTitle"))
				Expect(c.Address.Additional1).To(BeEquivalentTo("Additional1"))
				Expect(c.Address.Additional2).To(BeEquivalentTo("Additional2"))
				Expect(c.Type).To(BeEquivalentTo(2))
				Expect(c.Title).To(BeEquivalentTo("MegaHyperTestContact2"))
				Expect(c.Description).To(BeEquivalentTo("Desc"))
				Expect(c.Additional).To(BeEquivalentTo("Additional"))
				Expect(c.TripType).To(BeEquivalentTo(1))
				Expect(c.Address.GeoZones[0].OuterMinLat).To(BeEquivalentTo(0.1333371537))
				Expect(c.Address.GeoZones[0].OuterMinLon).To(BeEquivalentTo(-0.13365129242981638))
				Expect(c.Address.GeoZones[0].OuterMaxLat).To(BeEquivalentTo(0.133651291570))
				Expect(c.Address.GeoZones[0].OuterMaxLon).To(BeEquivalentTo(-0.33015374301))
				Expect(c.Address.GeoZones[0].Rectangle).ToNot(BeNil())
				Expect(c.Address.GeoZones[0].Rectangle.TopLeftLat).To(BeEquivalentTo(0.1333371537))
				Expect(c.Address.GeoZones[0].Rectangle.TopLeftLon).To(BeEquivalentTo(-0.13365129242981638))
				Expect(c.Address.GeoZones[0].Rectangle.BotRightLat).To(BeEquivalentTo(0.13365129157))
				Expect(c.Address.GeoZones[0].Rectangle.BotRightLon).To(BeEquivalentTo(-0.33015374301))
			})
			By("Only contact without address", func() {
				defer GinkgoRecover()
				var ctctId int64
				r := dbCon.QueryRow("Select _contactId from Contacts where title='MegaHyperTestContact2'")
				err := r.Scan(&ctctId)
				Expect(err).To(BeNil())
				c, err := GetContact(ctctId, dbCon, false)
				Expect(err).To(BeNil())
				Expect(c.Type).To(BeEquivalentTo(2))
				Expect(c.Title).To(BeEquivalentTo("MegaHyperTestContact2"))
				Expect(c.Description).To(BeEquivalentTo("Desc"))
				Expect(c.Additional).To(BeEquivalentTo("Additional"))
				Expect(c.TripType).To(BeEquivalentTo(1))
			})

			By("Only address without GeoZone", func() {
				defer GinkgoRecover()
				var addId int64
				r := dbCon.QueryRow("Select addressId from Contacts where title='MegaHyperTestContact2'")
				err := r.Scan(&addId)
				Expect(err).To(BeNil())
				a, err := GetAddress(addId, dbCon)
				Expect(err).To(BeNil())
				Expect(a.Id).To(BeNumerically(">", 0))
				Expect(a.Latitude).To(BeEquivalentTo(0.13333333337))
				Expect(a.Longitude).To(BeEquivalentTo(-0.13333333337))
				Expect(a.Street).To(BeEquivalentTo("Megahypererfundene Straße2"))
				Expect(a.Postal).To(BeEquivalentTo("01337"))
				Expect(a.City).To(BeEquivalentTo("Übelste City Oida"))
				Expect(a.Additional1).To(BeEquivalentTo("Additional1"))
				Expect(a.Additional2).To(BeEquivalentTo("Additional2"))
			})

			By("Get all addresses with GeoFences not crashing and containing something", func() { // Get contact tests most part of GetAddresses anyways
				defer GinkgoRecover()
				res, err := GetAddressesWithGeoFences("", dbCon)
				Expect(err).To(BeNil())
				Expect(len(res)).To(BeNumerically(">", 0))
			})
		})

		It("Should provide CRUD using JSON Api", func() {
			By("Read contact, including Address and GeoZones!", func() {
				defer GinkgoRecover()
				_, err := dbCon.Exec(`INSERT INTO "Addresses"(street,postal,city,additional1,additional2,latitude,longitude,HouseNumber,title)
			 VALUES('Megahypererfundene Straße3','01337','Übelste City Oida',
			'Additional1','Additional2',0.13333333337,-0.13333333337,'13','AddTitle');
			INSERT INTO "Rectangles" (topLeftLat,topLeftLon,botRightLat,botRightLon) VALUES(0.1333371538,-0.13365129242981638,0.13365129157,-0.33015374301);

INSERT INTO "GeoFenceRegions" (outerMinLat,outerMinLon,outerMaxLat,outerMaxLon,rectangleId,circleId) VALUES(0.1333371538,-0.13365129242981638,0.13365129157,-0.33015374301,
(SELECT _rectangleId FROM Rectangles WHERE topLeftLat = 0.1333371538 LIMIT 1),
NULL);
INSERT INTO Address_GeoFenceRegion (geoFenceRegionId,addressId) VALUES (
(SELECT _geoFenceRegionId FROM GeoFenceRegions WHERE outerMinLat = 0.1333371538 LIMIT 1),
(SELECT _addressId FROM Addresses where street='Megahypererfundene Straße3' LIMIT 1)
);
INSERT INTO "Contacts"(type,title,description,additional,addressId,tripTypeId)
 VALUES(2,'MegaHyperTestContact3','Desc','Additional',
(SELECT _addressId from Addresses where street='Megahypererfundene Straße3' LIMIT 1),
1);
			`)
				Expect(err).To(BeNil())
				var id int64
				row := dbCon.QueryRow("SELECT _contactId FROM Contacts WHERE title='MegaHyperTestContact3'")
				err = row.Scan(&id)
				Expect(err).To(BeNil())
				res, err := JSONSelectContact(`{"ID":`+strconv.FormatInt(id, 10)+`}`, dbCon)
				Expect(err).To(BeNil())

				Expect(res.ErrorMessage).To(BeEmpty())
				Expect(res.Success).To(BeTrue())
				Expect(res.Error).To(BeFalse())
				Expect(res.Errors).To(BeEmpty())
				Expect(res.Result).To(BeAssignableToTypeOf(&Contact{}))
				c := res.Result.(*Contact)
				Expect(c.Address.Id).To(BeNumerically(">", 0))
				Expect(len(c.Address.GeoZones)).To(BeEquivalentTo(1))
				Expect(c.Address.GeoZones[0].Id).To(BeNumerically(">", 0))
				Expect(c.Address.GeoZones[0].Rectangle).ToNot(BeNil())
				Expect(c.Address.GeoZones[0].Rectangle.Id).To(BeNumerically(">", 0))
				Expect(c.Address.Latitude).To(BeEquivalentTo(0.13333333337))
				Expect(c.Address.Longitude).To(BeEquivalentTo(-0.13333333337))
				Expect(c.Address.Street).To(BeEquivalentTo("Megahypererfundene Straße3"))
				Expect(c.Address.Postal).To(BeEquivalentTo("01337"))
				Expect(c.Address.City).To(BeEquivalentTo("Übelste City Oida"))
				Expect(c.Address.HouseNumber).To(BeEquivalentTo("13"))
				Expect(c.Address.Additional1).To(BeEquivalentTo("Additional1"))
				Expect(c.Address.Additional2).To(BeEquivalentTo("Additional2"))
				Expect(c.Address.Title).To(BeEquivalentTo("AddTitle"))

				Expect(c.Type).To(BeEquivalentTo(2))
				Expect(c.Title).To(BeEquivalentTo("MegaHyperTestContact3"))
				Expect(c.Description).To(BeEquivalentTo("Desc"))
				Expect(c.Additional).To(BeEquivalentTo("Additional"))
				Expect(c.TripType).To(BeEquivalentTo(1))
				Expect(c.Address.GeoZones[0].OuterMinLat).To(BeEquivalentTo(0.1333371538))
				Expect(c.Address.GeoZones[0].OuterMinLon).To(BeEquivalentTo(-0.13365129242981638))
				Expect(c.Address.GeoZones[0].OuterMaxLat).To(BeEquivalentTo(0.133651291570))
				Expect(c.Address.GeoZones[0].OuterMaxLon).To(BeEquivalentTo(-0.33015374301))
				Expect(c.Address.GeoZones[0].Rectangle).ToNot(BeNil())
				Expect(c.Address.GeoZones[0].Rectangle.TopLeftLat).To(BeEquivalentTo(0.1333371538))
				Expect(c.Address.GeoZones[0].Rectangle.TopLeftLon).To(BeEquivalentTo(-0.13365129242981638))
				Expect(c.Address.GeoZones[0].Rectangle.BotRightLat).To(BeEquivalentTo(0.13365129157))
				Expect(c.Address.GeoZones[0].Rectangle.BotRightLon).To(BeEquivalentTo(-0.33015374301))

				//j, _ := json.Marshal(c)
				//dbg.WTF(TAG, string(j))
			})
			By("Create contact, including Address and GeoZones (using GetContactWithGeoZone)!", func() {
				defer GinkgoRecover()
				jString := `{"Title":"MegaHyperTestContact4","Description":"Desc","Additional":"Additional","TripType":1,"Type":2,"Address":{"Latitude":0.13333333337,"Longitude":-0.13333333337,"Street":"Megahypererfundene Straße3","Postal":"01337","City":"Übelste City Oida","Additional1":"Additional1","Additional2":"Additional2","HouseNumber":"1A","Title":"AddTitle","Fuel":"ARAL"}}`
				res, err := JSONCreateContact(jString, dbCon)
				Expect(err).To(BeNil())

				Expect(res.ErrorMessage).To(BeEmpty())
				Expect(res.Success).To(BeTrue())
				Expect(res.Error).To(BeFalse())
				Expect(res.Errors).To(BeEmpty())
				Expect(res.LastKey).To(BeNumerically(">", 0))

				c, err := GetContactWithGeoZone(res.LastKey, dbCon)
				Expect(err).To(BeNil())
				j, _ := json.Marshal(c)
				Expect(c.Address).ToNot(BeNil())
				Expect(c.Address.GeoZones).ToNot(BeEmpty())
				Expect(c.Address.GeoZones[0].Rectangle).ToNot(BeNil())

				expected := `{"Id":` + strconv.FormatInt(res.LastKey, 10)
				expected += `,"Title":"MegaHyperTestContact4","Description":"Desc","Additional":"Additional","TripType":1,"Type":2,"Disabled":0`
				expected += `,"Address":{"Id":` + strconv.FormatInt(int64(c.Address.Id), 10) + `,"Latitude":0.13333333337,"Longitude":-0.13333333337,"Street":"Megahypererfundene Straße3","Postal":"01337","City":"Übelste City Oida","Additional1":"Additional1","Additional2":"Additional2","HouseNumber":"1A","Title":"AddTitle","Fuel":"ARAL"`
				expected += `,"GeoCoder":"","GeoZones":[{"Id":` + strconv.FormatInt(int64(c.Address.GeoZones[0].Id), 10) + `,"OuterMinLat":0.1330153751649422,"OuterMinLon":-0.13365129242981638,"OuterMaxLat":0.13365129157095162,"OuterMaxLon":-0.1330153743019036,"Color":"` + DefaultGeoZoneColor + `"`
				expected += `,"Rectangle":{"Id":` + strconv.FormatInt(int64(c.Address.GeoZones[0].Rectangle.Id), 10) + `,"TopLeftLat":0.1330153751649422,"TopLeftLon":-0.13365129242981638,"BotRightLat":0.13365129157095162,"BotRightLon":-0.1330153743019036}}]},"SyncedWith":""}`

				Expect(string(j)).To(BeEquivalentTo(expected))
			})

			By("Delete contact", func() {
				_, err := dbCon.Exec(`INSERT INTO "Addresses"(street,postal,city,additional1,additional2,latitude,longitude,HouseNumber)
			 VALUES('Megahypererfundene Straße5','01337','Übelste City Oida',
			'Additional1','Additional2',0.13333333337,-0.13333333337,'13');
			INSERT INTO "Rectangles" (topLeftLat,topLeftLon,botRightLat,botRightLon) VALUES(0.1333371538,-0.13365129242981638,0.13365129157,-0.33015374301);

INSERT INTO "GeoFenceRegions" (outerMinLat,outerMinLon,outerMaxLat,outerMaxLon,rectangleId,circleId) VALUES(0.1333371538,-0.13365129242981638,0.13365129157,-0.33015374301,
(SELECT _rectangleId FROM Rectangles WHERE topLeftLat = 0.1333371538 LIMIT 1),
NULL);
INSERT INTO Address_GeoFenceRegion (geoFenceRegionId,addressId) VALUES (
(SELECT _geoFenceRegionId FROM GeoFenceRegions WHERE outerMinLat = 0.1333371538 LIMIT 1),
(SELECT _addressId FROM Addresses where street='Megahypererfundene Straße5' LIMIT 1)
);
INSERT INTO "Contacts"(type,title,description,additional,addressId,tripTypeId)
 VALUES(2,'MegaHyperTestContact5','Desc','Additional',
(SELECT _addressId from Addresses where street='Megahypererfundene Straße5' LIMIT 1),
1);
			`)
				Expect(err).To(BeNil())
				var id int64
				row := dbCon.QueryRow("SELECT _contactId FROM Contacts WHERE title='MegaHyperTestContact5'")
				err = row.Scan(&id)
				Expect(err).To(BeNil())
				res, err := JSONDeleteContact("{\"ID\":"+strconv.FormatInt(id, 10)+"}", dbCon)
				Expect(res.ErrorMessage).To(BeEmpty())
				Expect(res.Success).To(BeTrue())
				Expect(res.Error).To(BeFalse())
				Expect(res.Errors).To(BeEmpty())
				Expect(res.RowCount).To(BeEquivalentTo(1))
				Expect(res.Id).To(BeEquivalentTo(id))
				row = dbCon.QueryRow("SELECT _contactId FROM Contacts WHERE title='MegaHyperTestContact5'")
				var i sql.NullInt64
				err = row.Scan(&i)

				Expect(err).To(BeEquivalentTo(sql.ErrNoRows))
			})

			By("Update contact (using GetContact)!", func() {
				_, err := dbCon.Exec(`INSERT INTO "Addresses"(street,postal,city,additional1,additional2,latitude,longitude,HouseNumber)
			 VALUES('Megahypererfundene Straße6','01337','Übelste City Oida',
			'Additional1','Additional2',0.13333333337,-0.13333333337,'13');
			INSERT INTO "Rectangles" (topLeftLat,topLeftLon,botRightLat,botRightLon) VALUES(0.1333371538,-0.13365129242981638,0.13365129157,-0.33015374301);

INSERT INTO "GeoFenceRegions" (outerMinLat,outerMinLon,outerMaxLat,outerMaxLon,rectangleId,circleId) VALUES(0.1333371538,-0.13365129242981638,0.13365129157,-0.33015374301,
(SELECT _rectangleId FROM Rectangles WHERE topLeftLat = 0.1333371538 LIMIT 1),
NULL);
INSERT INTO Address_GeoFenceRegion (geoFenceRegionId,addressId) VALUES (
(SELECT _geoFenceRegionId FROM GeoFenceRegions WHERE outerMinLat = 0.1333371538 LIMIT 1),
(SELECT _addressId FROM Addresses where street='Megahypererfundene Straße6' LIMIT 1)
);
INSERT INTO "Contacts"(type,title,description,additional,addressId,tripTypeId)
 VALUES(2,'MegaHyperTestContact6','Desc','Additional',
(SELECT _addressId from Addresses where street='Megahypererfundene Straße6' LIMIT 1),
1);
			`)
				Expect(err).To(BeNil())
				var id int64
				row := dbCon.QueryRow("SELECT _contactId FROM Contacts WHERE title='MegaHyperTestContact6'")
				err = row.Scan(&id)
				Expect(err).To(BeNil())
				res, err := JSONUpdateContact("{\"ID\":"+strconv.FormatInt(id, 10)+
				`,"Type":1,
	"Title":"MegaHyperTestContact7U",
	"Description":"DescU",
	"Additional":"AdditionalU",
	"TripType":2`+
				"}", dbCon)
				Expect(res.ErrorMessage).To(BeEmpty())
				Expect(res.Success).To(BeTrue())
				Expect(res.Error).To(BeFalse())
				Expect(res.Errors).To(BeEmpty())
				Expect(res.RowCount).To(BeEquivalentTo(1))
				Expect(res.Id).To(BeEquivalentTo(id))
				c, err := GetContact(id, dbCon, true)
				Expect(c.Type).To(BeEquivalentTo(1))
				Expect(c.Title).To(BeEquivalentTo("MegaHyperTestContact7U"))
				Expect(c.Description).To(BeEquivalentTo("DescU"))
				Expect(c.Additional).To(BeEquivalentTo("AdditionalU"))
				Expect(c.TripType).To(BeEquivalentTo(2))
			})

		})
	})
})
