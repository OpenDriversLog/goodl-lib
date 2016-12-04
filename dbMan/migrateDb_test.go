package dbMan_test

import (
	"database/sql"
	"os"

	"github.com/Compufreak345/dbg"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/OpenDriversLog/goodl-lib/dbMan"
	"github.com/OpenDriversLog/goodl-lib/tools"
)

const mdbtTag = "lib/dbman/migratetest"

var _ = Describe("MigrateDb", func() {

	var (
		basePath string
		dbPath   string
		dbCon    *sql.DB
	)

	BeforeEach(func() {
		basePath = "/go/src/github.com/OpenDriversLog/goodl-lib/test_integration/"
		dbPath = basePath + "old-test.db"
		dbCon, _ = sql.Open("SQLITE", dbPath)
	})

	AfterEach(func() {
		defer dbCon.Close()
	})

	Describe("ExecMigrations", func() {
		Context("on old-test.db", func() {
			It("should ", func() {
				defer GinkgoRecover()
				// _, err := dbMan.ExecMigrations(dbCon)
				// Expect(err).ToNot(HaveOccurred())
				// Expect(result).ToNot(Equal(nil))
				// Expect(result).Should(HaveKeyWithValue(2, "ProtoTestSpreewald"))

				// Expect(result).Should(HaveLen(3))
			})
		})

	})

	Describe("checkIfUpgradeNeeded", func() {
		Context("on old-test.db", func() {
			It("should ", func() {
				defer GinkgoRecover()
				src := basePath + "trackrecords-old.db"
				dst := basePath + "test-upgrade-from-old.db"
				if _, err := os.Stat(dst); err == nil {
					Expect(os.Remove(dst)).To(Succeed())
				}

				err := tools.CopyFile(src, dst)
				dbCon, _ = sql.Open("SQLITE", dst)
				dbg.V(mdbtTag, "path", dst)
				err = dbMan.CheckIfUpgradeNeeded(dbCon, -1, dst)
				Expect(err).ToNot(HaveOccurred())

				// Fail("rawrawraw")
				// Expect(result).Should(HaveKeyWithValue(2, "ProtoTestSpreewald"))

				// Expect(result).Should(HaveLen(3))
				//Expect(os.Chown(dst, 1000, 1000)).To(Succeed())
			})
		})

	})

})
