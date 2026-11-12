package quote

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/alexcao194/kairso/pkg/currency"
)

// All money is an int64 in the currency's minor unit; rates are basis points
// (1% = 100 bp) and quantities are thousandths (1 unit = 1000). Intermediate
// products go through big.Int so large amounts never overflow.

const (
	bpScale  = 10_000
	qtyScale = 1_000
)

// divRound divides num by den rounding half away from zero (num, den >= 0).
func divRound(num, den *big.Int) int64 {
	q, r := new(big.Int).QuoRem(num, den, new(big.Int))
	if new(big.Int).Mul(r, big.NewInt(2)).Cmp(den) >= 0 {
		q.Add(q, big.NewInt(1))
	}
	return q.Int64()
}

func lineTotal(qtyMilli, unitPrice int64, discountBp int32) int64 {
	num := new(big.Int).Mul(big.NewInt(qtyMilli), big.NewInt(unitPrice))
	num.Mul(num, big.NewInt(int64(bpScale-discountBp)))
	return divRound(num, big.NewInt(qtyScale*bpScale))
}

func percentOf(amount int64, bp int32) int64 {
	num := new(big.Int).Mul(big.NewInt(amount), big.NewInt(int64(bp)))
	return divRound(num, big.NewInt(bpScale))
}

type totals struct {
	Subtotal, Discount, Tax, Grand int64
}

func computeTotals(lines []int64, discountType string, discountValue int64, vatBp int32) totals {
	var t totals
	for _, l := range lines {
		t.Subtotal += l
	}
	if discountType == "amount" {
		t.Discount = min(discountValue, t.Subtotal)
	} else {
		t.Discount = percentOf(t.Subtotal, int32(discountValue))
	}
	taxable := t.Subtotal - t.Discount
	t.Tax = percentOf(taxable, vatBp)
	t.Grand = taxable + t.Tax
	return t
}

// ---- amount in words ----

var viDigits = []string{"không", "một", "hai", "ba", "bốn", "năm", "sáu", "bảy", "tám", "chín"}
var viScales = []string{"", "nghìn", "triệu", "tỷ", "nghìn tỷ", "triệu tỷ", "tỷ tỷ"}

func viGroup(n int, full bool) string {
	h, t, u := n/100, (n/10)%10, n%10
	var parts []string
	if h > 0 || full {
		parts = append(parts, viDigits[h], "trăm")
	}
	switch {
	case t == 0:
		if u > 0 {
			if h > 0 || full {
				parts = append(parts, "linh")
			}
			parts = append(parts, viDigits[u])
		}
	case t == 1:
		parts = append(parts, "mười")
		switch u {
		case 0:
		case 5:
			parts = append(parts, "lăm")
		default:
			parts = append(parts, viDigits[u])
		}
	default:
		parts = append(parts, viDigits[t], "mươi")
		switch u {
		case 0:
		case 1:
			parts = append(parts, "mốt")
		case 4:
			parts = append(parts, "tư")
		case 5:
			parts = append(parts, "lăm")
		default:
			parts = append(parts, viDigits[u])
		}
	}
	return strings.Join(parts, " ")
}

func viWords(n int64) string {
	if n == 0 {
		return "không"
	}
	var groups []int
	for n > 0 {
		groups = append(groups, int(n%1000))
		n /= 1000
	}
	var parts []string
	for i := len(groups) - 1; i >= 0; i-- {
		g := groups[i]
		if g == 0 {
			continue
		}
		full := i < len(groups)-1
		parts = append(parts, viGroup(g, full))
		if viScales[i] != "" {
			parts = append(parts, viScales[i])
		}
	}
	return strings.Join(parts, " ")
}

var enOnes = []string{"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine",
	"ten", "eleven", "twelve", "thirteen", "fourteen", "fifteen", "sixteen", "seventeen", "eighteen", "nineteen"}
var enTens = []string{"", "", "twenty", "thirty", "forty", "fifty", "sixty", "seventy", "eighty", "ninety"}
var enScales = []string{"", "thousand", "million", "billion", "trillion", "quadrillion", "quintillion"}

func enGroup(n int) string {
	var parts []string
	if n >= 100 {
		parts = append(parts, enOnes[n/100], "hundred")
		n %= 100
	}
	switch {
	case n >= 20:
		w := enTens[n/10]
		if n%10 > 0 {
			w += "-" + enOnes[n%10]
		}
		parts = append(parts, w)
	case n > 0:
		parts = append(parts, enOnes[n])
	}
	return strings.Join(parts, " ")
}

func enWords(n int64) string {
	if n == 0 {
		return "zero"
	}
	var groups []int
	for n > 0 {
		groups = append(groups, int(n%1000))
		n /= 1000
	}
	var parts []string
	for i := len(groups) - 1; i >= 0; i-- {
		if groups[i] == 0 {
			continue
		}
		parts = append(parts, enGroup(groups[i]))
		if enScales[i] != "" {
			parts = append(parts, enScales[i])
		}
	}
	return strings.Join(parts, " ")
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	return strings.ToUpper(string(r[0])) + string(r[1:])
}

// amountInWords spells out an amount (in minor units) in the quote language.
// A non-zero fractional part is written as a fraction, e.g. "and 50/100".
func amountInWords(amount int64, code, locale string) string {
	exp := currency.MinorUnits(code)
	scale := int64(1)
	for range exp {
		scale *= 10
	}
	major, minor := amount/scale, amount%scale
	name := currency.Name(code, locale)
	if locale == "en" {
		s := capitalize(enWords(major)) + " " + name
		if minor > 0 {
			s += fmt.Sprintf(" and %0*d/%d", exp, minor, scale)
		}
		return s
	}
	s := capitalize(viWords(major)) + " " + name
	if minor > 0 {
		s += fmt.Sprintf(" và %0*d/%d", exp, minor, scale)
	}
	return s
}
