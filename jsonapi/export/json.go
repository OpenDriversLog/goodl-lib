package export

import (
	"database/sql"
	"path/filepath"

	"github.com/Compufreak345/dbg"
	"github.com/OpenDriversLog/goodl-lib/tools"
	"github.com/OpenDriversLog/goodl-lib/jsonapi/notificationManager"
	"github.com/OpenDriversLog/goodl-lib/translate"
)

// JSONExport exports the trips in the given timeFrame with the given carId into a file in the given directory.
// format : currently only pdf
func JSONExport(format string, targetDir string, startTime int64, endTime int64, carId int64,activeNotifications *[]*notificationManager.Notification,T *translate.Translater, dbCon *sql.DB) (answer JSONExportAnswer, err error) {
	answer.Success = false

	if format == "pdf" {
		// TODO: Make timeconfig language dependent!
		timeConfig := tools.GetDefaultTimeConfig()
		var resPath string
		resPath, err = ExportToPdf(targetDir+"/exported/pdfs/", startTime, endTime, carId, timeConfig,activeNotifications,T, dbCon)
		resPath = "./protectedDownload/exported/pdfs/" + filepath.Base(resPath)
		if err != nil {
			dbg.E(TAG, "Error JSONExport/exporting to pdf : ", err)
			err = nil
			answer.ErrorMessage = "Internal server error"
			answer.Error = true
			return
		}
		answer.Success = true
		answer.ResPath = resPath
		return
	}

	answer.ErrorMessage = "Format not supported"
	answer.Error = true
	return
}
