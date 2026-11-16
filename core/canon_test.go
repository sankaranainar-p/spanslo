package core

import "testing"

func canon(t *testing.T, in string) string {
	t.Helper()
	out, err := Canonicalize([]byte(in))
	if err != nil {
		t.Fatalf("Canonicalize(%s): %v", in, err)
	}
	return string(out)
}

// R-005: number serialization (RFC 8785 section 3.2.2.3 and ES6 cases).
func TestCanonNumbers(t *testing.T) {
	for in, want := range map[string]string{
		"0":                      "0",
		"-0":                     "0",
		"1":                      "1",
		"-1.5":                   "-1.5",
		"4.50":                   "4.5",
		"2e-3":                   "0.002",
		"1e20":                   "100000000000000000000",
		"1e21":                   "1e+21",
		"1E30":                   "1e+30",
		"1e-6":                   "0.000001",
		"1e-7":                   "1e-7",
		"12e-8":                  "1.2e-7",
		"123456789012345680000":  "123456789012345680000",
		"9007199254740993":       "9007199254740992",
		"333333333.33333329":     "333333333.3333333",
		"5e-324":                 "5e-324",
		"1.7976931348623157e308": "1.7976931348623157e+308",
	} {
		if got := canon(t, in); got != want {
			t.Errorf("%s: got %s, want %s", in, got, want)
		}
	}
	if _, err := Canonicalize([]byte("1e999")); err == nil {
		t.Error("1e999: want error")
	}
}

// R-005: RFC 8785 section 3.2.3 example (key order, number and string forms).
func TestCanonRFCExample(t *testing.T) {
	in := `{"numbers":[333333333.33333329,1E30,4.50,2e-3,0.000000000000000000000000001],` +
		`"string":"\u20ac$\u000F\u000aA'\u0042\u0022\u005c\\\"\/","literals":[null,true,false]}`
	want := `{"literals":[null,true,false],"numbers":[333333333.3333333,1e+30,4.5,0.002,1e-27],` +
		`"string":"€$\u000f\nA'B\"\\\\\"/"}`
	if got := canon(t, in); got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

// R-005: keys sort by UTF-16 code units (RFC 8785 section 3.2.3 sorting example).
func TestCanonKeyOrder(t *testing.T) {
	in := `{"\u20ac":"Euro","\r":"CR","\ufb33":"Dalet","1":"One","\ud83d\ude00":"Emoji","\u0080":"Ctl","\u00f6":"O"}`
	want := "{\"\\r\":\"CR\",\"1\":\"One\",\"\u0080\":\"Ctl\",\"\u00f6\":\"O\",\"\u20ac\":\"Euro\"," +
		"\"\U0001F600\":\"Emoji\",\"\ufb33\":\"Dalet\"}"
	if got := canon(t, in); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

// R-005: string escaping and unicode passthrough.
func TestCanonStrings(t *testing.T) {
	got := canon(t, `"\u0000\u001f\b\t\n\f\r\"\\\u003c\u2028é"`)
	want := "\"\\u0000\\u001f\\b\\t\\n\\f\\r\\\"\\\\<\u2028é\""
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCanonRejectsBadInput(t *testing.T) {
	for _, in := range []string{"", "{", `{"a":1} x`, `{"a":1}{"b":2}`} {
		if _, err := Canonicalize([]byte(in)); err == nil {
			t.Errorf("%q: want error", in)
		}
	}
}

func TestArgsHash(t *testing.T) {
	h, err := ArgsHash([]byte(" { } "))
	if err != nil || h != "44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a" {
		t.Fatalf("empty object: %q, %v", h, err)
	}
	a, _ := ArgsHash([]byte(`{"b":1.0,"a":[1,2]}`))
	b, _ := ArgsHash([]byte(`{ "a":[1,2], "b":1 }`))
	c, _ := ArgsHash([]byte(`{"a":[2,1],"b":1}`))
	if a != b || a == c {
		t.Errorf("hash equality wrong: %s %s %s", a, b, c)
	}
	if _, err := ArgsHash([]byte("{")); err == nil {
		t.Error("want error on invalid JSON")
	}
}
