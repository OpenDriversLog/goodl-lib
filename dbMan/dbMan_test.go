package dbMan_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/OpenDriversLog/goodl-lib/dbMan"
)

var _ = Describe("DbMan", func() {

	var (
		basePath string
		dbPath   string
	// dbCon    *sql.DB
	)

	BeforeEach(func() {
		basePath = "/go/src/github.com/OpenDriversLog/goodl-lib/test_integration/"
		dbPath = basePath + "trackrecords.db"
		// dbCon, _ = sql.Open("SQLITE3", dbPath)
	})

	AfterEach(func() {
		// defer dbCon.Close()
	})

	Describe("CreateNewLocationDb", func() {
		Context("path w/ no file", func() {
			It("should be able to create one with latest migration", func() {
				defer GinkgoRecover()
				err := dbMan.CreateNewLocationDb(basePath + "newCreate.db")
				Expect(err).ToNot(HaveOccurred())
				Expect(os.Remove(basePath + "newCreate.db")).To(Succeed())
			})
		})
	})

	Describe("GetLocatoinDb", func() {
		Context("path w/ no file", func() {
			It("should be able to create one with latest migration", func() {
				defer GinkgoRecover()
				dbCon2, err := dbMan.GetLocationDb(basePath + "newGetLoc.db",-1)
				Expect(err).ToNot(HaveOccurred())
				Expect(dbCon2.Ping()).To(Succeed())
				Expect(os.Remove(basePath + "newGetLoc.db")).To(Succeed())
			})
		})

		Context("path w/ up-to-date file", func() {
			It("should be able connect & query DB", func() {
				defer GinkgoRecover()
				dbCon2, err := dbMan.GetLocationDb(basePath + "tests.db",-1)
				Expect(err).ToNot(HaveOccurred())
				Expect(dbCon2.Ping()).To(Succeed())
			})
		})
	})

})
