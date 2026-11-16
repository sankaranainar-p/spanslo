package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode/utf16"
)

// Canonicalize returns the RFC 8785 (JSON Canonicalization Scheme) form of one JSON
// value. Numbers go through float64, as the RFC requires (R-005).
// ponytail: duplicate object keys keep the last value; the RFC says reject them.
func Canonicalize(data []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, fmt.Errorf("canonicalize: trailing data after JSON value")
	}
	var b bytes.Buffer
	if err := writeCanon(&b, v); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// ArgsHash returns the lowercase hex SHA-256 of the canonical form of data. Hex strings
// of equal length order the same as the raw bytes (R-005).
func ArgsHash(data []byte) (string, error) {
	c, err := Canonicalize(data)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(c)
	return hex.EncodeToString(sum[:]), nil
}

func writeCanon(b *bytes.Buffer, v any) error {
	switch x := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		b.WriteString(strconv.FormatBool(x))
	case string:
		writeCanonString(b, x)
	case json.Number:
		f, err := strconv.ParseFloat(x.String(), 64)
		if err != nil || math.IsInf(f, 0) {
			return fmt.Errorf("canonicalize: number %s out of range", x)
		}
		b.WriteString(formatES6(f))
	case []any:
		b.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				b.WriteByte(',')
			}
			if err := writeCanon(b, e); err != nil {
				return err
			}
		}
		b.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		// RFC 8785 sorts keys by UTF-16 code units.
		slices.SortFunc(keys, func(a, c string) int {
			return slices.Compare(utf16.Encode([]rune(a)), utf16.Encode([]rune(c)))
		})
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			writeCanonString(b, k)
			b.WriteByte(':')
			if err := writeCanon(b, x[k]); err != nil {
				return err
			}
		}
		b.WriteByte('}')
	default:
		return fmt.Errorf("canonicalize: unexpected type %T", v)
	}
	return nil
}

var escapes = map[rune]string{'"': `\"`, '\\': `\\`, '\b': `\b`, '\t': `\t`, '\n': `\n`, '\f': `\f`, '\r': `\r`}

func writeCanonString(b *bytes.Buffer, s string) {
	b.WriteByte('"')
	for _, r := range s {
		if e, ok := escapes[r]; ok {
			b.WriteString(e)
		} else if r < 0x20 {
			fmt.Fprintf(b, `\u%04x`, r)
		} else {
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
}

// formatES6 formats f like ECMAScript Number.prototype.toString (RFC 8785 section 3.2.2.3),
// using Go's shortest round-trip digits.
func formatES6(f float64) string {
	if f == 0 {
		return "0" // also -0
	}
	if f < 0 {
		return "-" + formatES6(-f)
	}
	s := strconv.FormatFloat(f, 'e', -1, 64) // d.ddde±XX
	mant, expStr, _ := strings.Cut(s, "e")
	exp, _ := strconv.Atoi(expStr)
	digits := strings.Replace(mant, ".", "", 1)
	k, n := len(digits), exp+1
	switch {
	case k <= n && n <= 21:
		return digits + strings.Repeat("0", n-k)
	case 0 < n && n <= 21:
		return digits[:n] + "." + digits[n:]
	case -6 < n && n <= 0:
		return "0." + strings.Repeat("0", -n) + digits
	}
	e := n - 1
	sign := "+"
	if e < 0 {
		sign, e = "-", -e
	}
	if k == 1 {
		return digits + "e" + sign + strconv.Itoa(e)
	}
	return digits[:1] + "." + digits[1:] + "e" + sign + strconv.Itoa(e)
}
