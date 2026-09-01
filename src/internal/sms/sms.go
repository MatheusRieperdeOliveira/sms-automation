package sms

import "strings"

var normalize = strings.NewReplacer(
	".", "",
	",", ".",
	";", ",",
) 

func Normalize (sms string) string {
	return normalize.Replace(sms)
}
