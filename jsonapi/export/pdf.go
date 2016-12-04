// Package export is responsible for exporting trips to pdf-format.
package export
/**
TODO: Make multilingual!!
 */
import (
	"database/sql"
	"os"
	"strings"

	"github.com/jung-kurt/gofpdf"

	"github.com/Compufreak345/dbg"
	"github.com/qiniu/iconv"
	"github.com/OpenDriversLog/goodl-lib/jsonapi/carManager"
	"github.com/OpenDriversLog/goodl-lib/jsonapi/driverManager"
	"github.com/OpenDriversLog/goodl-lib/jsonapi/tripMan"
	"github.com/OpenDriversLog/goodl-lib/tools"
	"github.com/OpenDriversLog/goodl-lib/jsonapi/notificationManager"
	"github.com/OpenDriversLog/goodl-lib/translate"
	"github.com/OpenDriversLog/goodl-lib/jsonapi/tripMan/models"
	"fmt"
	"time"
	"github.com/OpenDriversLog/goodl-lib/jsonapi/addressManager"
	S "github.com/OpenDriversLog/goodl-lib/models/SQLite"

)

var TAG = dbg.Tag("goodl-lib/jsonApi/export")
type Row struct {
	W float64
	H float64
	Cells []*Cell
}

type Cell struct {
	W float64
	H float64
	Val string
	Color Color
}

type Color struct {
	R int
	G int
	B int
}


// ExportToPdf exports the trips in the given timespan for the given car to a file in the given directory,
// returning the path of the result file.
func ExportToPdf(dir string, startTime int64, endTime int64, carId int64, timeConfig *tools.TimeConfig,activeNotifications *[]*notificationManager.Notification,T *translate.Translater, dbCon *sql.DB) (resPath string, err error) {
	if dir == "" {
		dbg.E(TAG, "Error path for pdf was empty : ", err)
		return
	}

	os.MkdirAll(dir, 0755)
	fName := "Fahrtenbuch_" + tools.GetDateForFileName(startTime, timeConfig) + " - " + tools.GetDateForFileName(endTime, timeConfig) + ".pdf"
	resPath, err = tools.GetCleanFilePath(fName, dir)
	if err != nil {
		dbg.E(TAG, "Error getting pdf file path : ", err)
		return
	}
	pdf := gofpdf.New("L", "mm", "A4", "")
	// Column widths
	w := []float64{25.0, 30.0, 20.0, 20.0, 35.0, 35.0, 35.0, 25.0, 55.0}
	wSum := 0.0
	header := []string{"Datum", "Art der Fahrt", "km Start", "km Ende", "Start", "Ziel", "Kundenadresse", "Fahrer", "Grund"}

	trips, err := tripMan.GetTripsByWhere("sEndTime<=? AND eStartTime>=? AND sCarId=?", true, false, true, activeNotifications,T,true, dbCon, endTime, startTime, carId)
	drivers, err := driverManager.GetDrivers(dbCon)
	contacts, err := addressManager.GetContactsWithGeoZones("",dbCon)
	if err != nil {
		dbg.E(TAG, "Error getting drivers!", err)
		return
	}
	dById := make(map[int64]*driverManager.Driver)
	for _, v := range drivers {
		dById[int64(v.Id)] = v
	}
	if err != nil {
		dbg.E(TAG, "Error getting trips : ", err)
		return
	}
	// Data
	maxY := 0.0
	minY := 0.0
	firstPage := true
	tlen := len(trips)
	rows := make([]*Row,0)
	if tlen == 0 {
		pdf.AddPage()
		pdf.SetFont("Arial", "B", 16)
		pdf.Cell(100, 20, "Automagisch Fahrtentracker - Fahrtenbuch")
		pdf.Ln(-1)
		pdf.SetFont("Arial", "", 14)
		var car *carManager.Car
		car, err = carManager.GetCarById(dbCon, carId)
		if err != nil {
			dbg.E(TAG, "Unable to get car with Id %d : %s", carId, err)
			return
		}
		pdf.Cell(200, 15, convertUtfToIso("Es sind im ausgewählten Zeitraum keine Fahrten für das Fahrzeug "+string(car.Plate)+" verfügbar."))
	} else {
		pdf.AddPage();
		pdf.SetFont("Arial", "", 12)
		for _,t := range trips {
			maxH := 0.0
			r := Row{ Cells: make([]*Cell,0)}

			dateString := tools.GetDateOnlyForText(int64(t.StartTime), true) + "-"
			tstart := tools.GetTimeFromMillis(int64(t.StartTime))
			tend := tools.GetTimeFromMillis(int64(t.EndTime))
			if tstart.Day() == tend.Day() && tstart.Month() == tend.Month() && tend.Year() == tstart.Year() {
				dateString += tools.GetDateOnlyForText(int64(t.EndTime), false)
			} else {
				dateString += tools.GetDateOnlyForText(int64(t.EndTime), true)
			}
			r.Cells = append(r.Cells,calcMultiCell(w[0], dateString, pdf, &maxH))

			ttype := "???"

			var col Color
			switch int(t.Type) {
			case 1:
				ttype = "Privat"
				col = Color{R:100,G:0,B:0}
			case 2:
				ttype = "Arbeitsweg"
				col = Color{R:0,G:0,B:100}
			case 3:
				ttype = "Dienstlich"
				col = Color{R:0,G:100,B:0}
			}
			c := calcMultiCell(w[1], getValWithHistory(ttype, "Type",t.History,drivers,contacts), pdf, &maxH)
			c.Color = col
			r.Cells = append(r.Cells,c)
			pdf.SetTextColor(0, 0, 0)
			r.Cells = append(r.Cells,calcMultiCell(w[2], getValWithHistory("???", "???",t.History,drivers,contacts),
				pdf, &maxH))
			r.Cells = append(r.Cells,calcMultiCell(w[3], getValWithHistory("???", "???",t.History,drivers,contacts),
				pdf, &maxH))
			if t.Type != 1 {
				r.Cells = append(r.Cells,calcMultiCell(w[4], strings.Join([]string{
					string(t.StartAddress.Postal) + " " + string(t.StartAddress.City),
					string(t.StartAddress.Street) + " " + string(t.StartAddress.HouseNumber),
				}, "\n"),
					pdf, &maxH))
				r.Cells = append(r.Cells,calcMultiCell(w[5], strings.Join([]string{
					string(t.EndAddress.Postal) + " " + string(t.EndAddress.City),
					string(t.EndAddress.Street) + " " + string(t.EndAddress.HouseNumber),
				}, "\n"), pdf, &maxH))
				if t.EndContact != nil {

					r.Cells = append(r.Cells,calcMultiCell(w[6], getValWithHistory(strings.Join([]string{
						string(t.EndContact.Address.Postal) + " " + string(t.EndContact.Address.City),
						string(t.EndContact.Address.Street) + " " + string(t.EndContact.Address.HouseNumber),
					}, "\n"),"EndContactId",t.History,drivers,contacts),
						pdf, &maxH))
				} else {
					r.Cells = append(r.Cells,calcMultiCell(w[6], getValWithHistory("", "EndContactId",t.History,drivers,contacts),
						pdf, &maxH))
					dbg.W(TAG, "No EndContact for trip ", t.Id)
				}
				var driver *driverManager.Driver
				if int64(t.DriverId) != 0 {
					driver = dById[int64(t.DriverId)]
				}
				if driver == nil {
					r.Cells = append(r.Cells,calcMultiCell(w[7], getValWithHistory("Unbekannt", "DriverId",t.History,drivers,contacts),
						pdf, &maxH))
				} else {
					r.Cells = append(r.Cells,calcMultiCell(w[7], getValWithHistory(string(driver.Name), "DriverId",t.History,drivers,contacts),
						pdf, &maxH))
				}
				desc := string(t.Description)
				if len(t.TrackDetails) > 1 {
					desc += "\nZwischenstopps:\n"
					for c := 1; c < len(t.TrackDetails); c++ {
						tr := t.TrackDetails[c]
						if c > 1 {
							desc += "\n"
						}
						desc += string(tr.StartKeyPointInfo.Postal) + " " + string(tr.StartKeyPointInfo.City) + "," +
							string(tr.StartKeyPointInfo.Street) + " " + string(tr.StartKeyPointInfo.HouseNumber)

					}
				}
				r.Cells = append(r.Cells,calcMultiCell(w[8], getValWithHistory(string(desc), "Description",t.History,drivers,contacts),
					pdf, &maxH))
			} else {
				r.Cells = append(r.Cells,calcMultiCell(w[4], "",
					pdf, &maxH))
				r.Cells = append(r.Cells,calcMultiCell(w[5], "",
					pdf, &maxH))
				r.Cells = append(r.Cells,calcMultiCell(w[6], "",
					pdf, &maxH))
				r.Cells = append(r.Cells,calcMultiCell(w[7], "",
					pdf, &maxH))
				r.Cells = append(r.Cells,calcMultiCell(w[8], "",
					pdf, &maxH))
			}
			r.H = maxH
			rows = append(rows,&r)
		}
		startX := 0.0
		for i,r := range rows {
			if maxY + r.H > 190.0 || firstPage { // Starting a new page (maxY==0.0 is only true on first page)
				if !firstPage {
					pdf.AddPage()
				} else {
					firstPage = false
				}
				pdf.SetFont("Arial", "B", 16)
				pdf.Cell(100, 20, "Automagisch Fahrtentracker - Fahrtenbuch")
				pdf.Ln(-1)
				pdf.SetFont("Arial", "", 14)
				var car *carManager.Car
				car, err = carManager.GetCarById(dbCon, carId)
				if err != nil {
					dbg.E(TAG, "Unable to get car with Id %d : %s", carId, err)
					return
				}
				pdf.Cell(200, 15, convertUtfToIso("Kfz: "+string(car.Plate)))
				pdf.Ln(-1)
				pdf.SetFont("Arial", "", 12)
				// 	Header
				for j, str := range header {
					pdf.CellFormat(w[j], 7, str, "1", 0, "C", false, 0, "")
				}
				pdf.Ln(-1)

				minY = pdf.GetY()
				maxY = minY + 2


				//Draw borders around table
				x := pdf.GetX()
				startX = x
				sx := x
				for j, _ := range header { // lines between cells
					sx += w[j]
				}
				ySum := maxY
				for j:=i;j<len(rows);j++ { // lines between rows
					lh := rows[j].H
					ns := ySum +lh
					if ns > 190.0 {
						break
					}
					ySum = ns

					pdf.Line(x, ySum-1, sx, ySum-1)
				}
				sx=x
				for j, _ := range header { // lines between cells
					pdf.Line(sx, minY, sx, ySum-1)
					sx += w[j]
				}

				pdf.Line(sx, minY, sx, ySum-1)
			}

			wSum = startX
			pdf.SetY(maxY)
			for k := 0;k<len(r.Cells);k++ {
				c := r.Cells[k]
				pdf.SetTextColor(c.Color.R,c.Color.G,c.Color.B)
				addMultiCell(w[k], c.Val, &wSum, pdf)
			}
			maxY += r.H
		}
	}
	//pdf.CellFormat(wSum, 0, "", "T", 0, "", false, 0, "")
	err = pdf.OutputFileAndClose(resPath)
	if err != nil {
		dbg.E(TAG, "Error writing pdf file : ", err)
		return
	}
	return
}

// getValWithHistory adds the historical changes for the given value to the "val"-string, returning it as newVal.
func getValWithHistory(val string, name string, history []*models.CleanTripHistoryEntry, drivers []*driverManager.Driver, contacts[]*addressManager.Contact) (newVal string){
	newVal = val
	if history == nil {
		return
	}
	newVal = "Aktuell : " + val
	for _,v := range history {
		for k,c := range v.Changes {
			if k == name {
				if  c.OldVal != c.NewVal {
					date,err := time.Parse(time.RFC3339, string(v.ChangeDate))
					var timeMillis int64
					if err != nil {
						dbg.WTF(TAG,"Error parsing ChangeDate %s : %s", v.ChangeDate, err)
					} else {
						timeMillis = date.Unix()*1000
					}
					if strings.Contains(k,"ContactId") {
						var id int64
						if c.NewVal != nil {
							id = int64(c.NewVal.(S.NInt64))
						}
						for _,cont:=range contacts {
							if int64(cont.Id) == id {
								c.NewVal = cont.Title
							}
						}
					} else if strings.Contains(k,"riverId") {
						var id int64
						if c.NewVal != nil {
							id = int64(c.NewVal.(S.NInt64))
						}
						for _,d:=range drivers {
							if int64(d.Id) == id {
								c.NewVal = d.Name
							}
						}
					} else if k=="Type" {
						switch int(c.NewVal.(S.NInt64)) {
						case 1:
							c.NewVal = "Privat"
						case 2:
							c.NewVal = "Arbeitsweg"
						case 3:
							c.NewVal = "Dienstlich"
						}
					}

					newVal += fmt.Sprintf("\r\n Ab %v : %v",tools.GetDateOnlyForText_NoTime(timeMillis),c.NewVal)
				}
			}
		}
	}
	return
}

// addMultiCell adds a multipline line-cell to the pdf, automatically splitting the text into lines to fit.
func addMultiCell(width float64, txt string, wSum *float64, pdf *gofpdf.Fpdf) {

	y := pdf.GetY()
	startY := y

	splitted := pdf.SplitLines([]byte(txt), width)

	for _, v := range splitted {
		pdf.SetXY(*wSum, y)
		pdf.CellFormat(width, 4, convertUtfToIso(string(v)), "", 0, "LB", false, 0, "")
		y += 4
	}
	//pdf.MultiCell(width,4,txt, "LR", "", false)

	*wSum += width
	pdf.SetXY(*wSum, startY)
	return
}

// calcMultiCell calculates the height of a multiline-cell resulting of the given text & width, limited by a maximum height maxH
func calcMultiCell(width float64, txt string,pdf *gofpdf.Fpdf, maxH *float64) (*Cell) {
	h:= float64(4*len(pdf.SplitLines([]byte(txt), width))) + 2
	if *maxH<h{
		*maxH = h
	}

	return &Cell {
		H:h,
		W:width,
		Val:txt,
	}
}

// convertUtfToIso Converts an UTF-8-String to an ISO-8859-1-string.
func convertUtfToIso(s string) (cs string) {
	cd, err := iconv.Open("ISO-8859-1", "utf-8") // convert utf-8 to gbk
	if err != nil {
		dbg.E(TAG, "iconv.Open failed!")
		return s
	}
	defer cd.Close()

	cs = cd.ConvString(s)

	return
}
