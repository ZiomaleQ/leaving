package main

import (
	"net/url"
	"strings"
	"unicode"
)

func decodeFromCda(code string) string {
	code = strings.Replace(code, "_XDDD", "", 1)
	code = strings.Replace(code, "_CDA", "", 1)
	code = strings.Replace(code, "_ADC", "", 1)
	code = strings.Replace(code, "_CXD", "", 1)
	code = strings.Replace(code, "_QWE", "", 1)
	code = strings.Replace(code, "_Q5", "", 1)
	code = strings.Replace(code, "_IKSDE", "", 1)

	code, _ = url.QueryUnescape(code)

	code = strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			charVal := int(r)
			if 33 <= charVal && 126 >= charVal {
				return rune(33 + ((charVal + 14) % 94))
			}
		}
		return r
	}, code)

	code = strings.Replace(code, ".cda.mp4", "", 1)
	code = strings.Replace(code, ".2cda.pl", ".cda.pl", 1)
	code = strings.Replace(code, ".3cda.pl", ".cda.pl", 1)

	return "https://" + strings.Replace(code, "/upstream", ".mp4/upstream", 1) + ".mp4"
}
