// Package currency lists the circulating ISO 4217 currencies and their minor
// unit exponents. Amounts in Kairso are stored as integers in the currency's
// minor unit (e.g. cents), so the exponent is needed to read and format them.
package currency

import "sort"

// codes are the circulating ISO 4217 currencies (fund, precious-metal and test
// codes are excluded since they are never quoted to customers).
var codes = []string{
	"AED", "AFN", "ALL", "AMD", "ANG", "AOA", "ARS", "AUD", "AWG", "AZN",
	"BAM", "BBD", "BDT", "BGN", "BHD", "BIF", "BMD", "BND", "BOB", "BRL",
	"BSD", "BTN", "BWP", "BYN", "BZD", "CAD", "CDF", "CHF", "CLP", "CNY",
	"COP", "CRC", "CUP", "CVE", "CZK", "DJF", "DKK", "DOP", "DZD", "EGP",
	"ERN", "ETB", "EUR", "FJD", "FKP", "GBP", "GEL", "GHS", "GIP", "GMD",
	"GNF", "GTQ", "GYD", "HKD", "HNL", "HTG", "HUF", "IDR", "ILS", "INR",
	"IQD", "IRR", "ISK", "JMD", "JOD", "JPY", "KES", "KGS", "KHR", "KMF",
	"KPW", "KRW", "KWD", "KYD", "KZT", "LAK", "LBP", "LKR", "LRD", "LSL",
	"LYD", "MAD", "MDL", "MGA", "MKD", "MMK", "MNT", "MOP", "MRU", "MUR",
	"MVR", "MWK", "MXN", "MYR", "MZN", "NAD", "NGN", "NIO", "NOK", "NPR",
	"NZD", "OMR", "PAB", "PEN", "PGK", "PHP", "PKR", "PLN", "PYG", "QAR",
	"RON", "RSD", "RUB", "RWF", "SAR", "SBD", "SCR", "SDG", "SEK", "SGD",
	"SHP", "SLE", "SOS", "SRD", "SSP", "STN", "SVC", "SYP", "SZL", "THB",
	"TJS", "TMT", "TND", "TOP", "TRY", "TTD", "TWD", "TZS", "UAH", "UGX",
	"USD", "UYU", "UZS", "VED", "VES", "VND", "VUV", "WST", "XAF", "XCD",
	"XCG", "XOF", "XPF", "YER", "ZAR", "ZMW", "ZWG",
}

// exponents lists the currencies whose minor unit is not 2.
var exponents = map[string]int{
	"BIF": 0, "CLP": 0, "DJF": 0, "GNF": 0, "ISK": 0, "JPY": 0, "KMF": 0,
	"KRW": 0, "PYG": 0, "RWF": 0, "UGX": 0, "VND": 0, "VUV": 0, "XAF": 0,
	"XOF": 0, "XPF": 0,
	"BHD": 3, "IQD": 3, "JOD": 3, "KWD": 3, "LYD": 3, "OMR": 3, "TND": 3,
}

var known = func() map[string]bool {
	m := make(map[string]bool, len(codes))
	for _, c := range codes {
		m[c] = true
	}
	return m
}()

// Currency describes one ISO 4217 currency.
type Currency struct {
	Code       string `json:"code"`
	MinorUnits int    `json:"minorUnits"`
}

// IsValid reports whether code is a circulating ISO 4217 currency.
func IsValid(code string) bool { return known[code] }

// MinorUnits returns the number of decimal digits of the currency's minor unit.
func MinorUnits(code string) int {
	if e, ok := exponents[code]; ok {
		return e
	}
	return 2
}

// All returns every supported currency, sorted by code.
func All() []Currency {
	out := make([]Currency, 0, len(codes))
	for _, c := range codes {
		out = append(out, Currency{Code: c, MinorUnits: MinorUnits(c)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out
}

var namesVI = map[string]string{
	"VND": "đồng", "USD": "đô la Mỹ", "EUR": "euro", "JPY": "yên Nhật", "CNY": "nhân dân tệ",
	"KRW": "won Hàn Quốc", "GBP": "bảng Anh", "AUD": "đô la Úc", "CAD": "đô la Canada",
	"SGD": "đô la Singapore", "HKD": "đô la Hồng Kông", "TWD": "Đài tệ", "THB": "baht Thái",
	"LAK": "kíp Lào", "KHR": "riel Campuchia", "MYR": "ringgit Malaysia", "IDR": "rupiah Indonesia",
	"PHP": "peso Philippines", "INR": "rupee Ấn Độ", "CHF": "franc Thụy Sĩ", "NZD": "đô la New Zealand",
	"RUB": "rúp Nga", "AED": "dirham UAE", "SAR": "riyal Ả Rập Xê Út", "SEK": "krona Thụy Điển",
	"NOK": "krone Na Uy", "DKK": "krone Đan Mạch", "MMK": "kyat Myanmar",
}

var namesEN = map[string]string{
	"VND": "Vietnamese dong", "USD": "US dollars", "EUR": "euros", "JPY": "Japanese yen",
	"CNY": "Chinese yuan", "KRW": "South Korean won", "GBP": "British pounds", "AUD": "Australian dollars",
	"CAD": "Canadian dollars", "SGD": "Singapore dollars", "HKD": "Hong Kong dollars",
	"TWD": "New Taiwan dollars", "THB": "Thai baht", "LAK": "Lao kip", "KHR": "Cambodian riels",
	"MYR": "Malaysian ringgit", "IDR": "Indonesian rupiah", "PHP": "Philippine pesos",
	"INR": "Indian rupees", "CHF": "Swiss francs", "NZD": "New Zealand dollars", "RUB": "Russian rubles",
	"AED": "UAE dirhams", "SAR": "Saudi riyals", "SEK": "Swedish kronor", "NOK": "Norwegian kroner",
	"DKK": "Danish kroner", "MMK": "Myanmar kyats",
}

// Name returns a human currency name for amount-in-words, falling back to the code.
func Name(code, locale string) string {
	table := namesVI
	if locale == "en" {
		table = namesEN
	}
	if n, ok := table[code]; ok {
		return n
	}
	return code
}
