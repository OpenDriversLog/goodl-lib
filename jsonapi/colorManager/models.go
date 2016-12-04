package colorManager

import (
	S "github.com/OpenDriversLog/goodl-lib/models/SQLite"
)

type Color struct {
	Id S.NInt64
	Color1 S.NString
	Color2 S.NString
	Color3 S.NString
}