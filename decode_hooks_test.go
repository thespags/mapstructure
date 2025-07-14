package mapstructure

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/netip"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

type decodeHookTestSuite[F any, T any] struct {
	fn   DecodeHookFunc
	ok   []decodeHookTestCase[F, T]
	fail []decodeHookFailureTestCase[F, T]
}

func (ts decodeHookTestSuite[F, T]) Run(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		for _, tc := range ts.ok {
			tc := tc

			t.Run("", func(t *testing.T) {
				t.Parallel()

				tc.Run(t, ts.fn)
			})
		}
	})

	t.Run("Fail", func(t *testing.T) {
		t.Parallel()

		for _, tc := range ts.ok {
			tc := tc

			t.Run("", func(t *testing.T) {
				t.Parallel()

				tc.Run(t, ts.fn)
			})
		}
	})

	t.Run("NoOp", func(t *testing.T) {
		t.Parallel()

		var zero F

		actual, err := DecodeHookExec(ts.fn, reflect.ValueOf(zero), reflect.ValueOf(zero))
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		if !reflect.DeepEqual(actual, zero) {
			t.Fatalf("expected %[1]T(%#[1]v), got %[2]T(%#[2]v)", zero, actual)
		}
	})
}

type decodeHookTestCase[F any, T any] struct {
	from     F
	expected T
}

func (tc decodeHookTestCase[F, T]) Run(t *testing.T, fn DecodeHookFunc) {
	var to T

	actual, err := DecodeHookExec(fn, reflect.ValueOf(tc.from), reflect.ValueOf(to))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if !reflect.DeepEqual(actual, tc.expected) {
		t.Fatalf("expected %[1]T(%#[1]v), got %[2]T(%#[2]v)", tc.expected, actual)
	}
}

type decodeHookFailureTestCase[F any, T any] struct {
	from F
}

func (tc decodeHookFailureTestCase[F, T]) Run(t *testing.T, fn DecodeHookFunc) {
	var to T

	_, err := DecodeHookExec(fn, reflect.ValueOf(tc.from), reflect.ValueOf(to))
	if err == nil {
		t.Fatalf("expected error, got none")
	}
}

func TestComposeDecodeHookFunc(t *testing.T) {
	f1 := func(
		f reflect.Kind,
		t reflect.Kind,
		data any,
	) (any, error) {
		return data.(string) + "foo", nil
	}

	f2 := func(
		f reflect.Kind,
		t reflect.Kind,
		data any,
	) (any, error) {
		return data.(string) + "bar", nil
	}

	f := ComposeDecodeHookFunc(f1, f2)

	result, err := DecodeHookExec(
		f, reflect.ValueOf(""), reflect.ValueOf([]byte("")))
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if result.(string) != "foobar" {
		t.Fatalf("bad: %#v", result)
	}
}

func TestComposeDecodeHookFunc_err(t *testing.T) {
	f1 := func(reflect.Kind, reflect.Kind, any) (any, error) {
		return nil, errors.New("foo")
	}

	f2 := func(reflect.Kind, reflect.Kind, any) (any, error) {
		panic("NOPE")
	}

	f := ComposeDecodeHookFunc(f1, f2)

	_, err := DecodeHookExec(
		f, reflect.ValueOf(""), reflect.ValueOf([]byte("")))
	if err.Error() != "foo" {
		t.Fatalf("bad: %s", err)
	}
}

func TestComposeDecodeHookFunc_kinds(t *testing.T) {
	var f2From reflect.Kind

	f1 := func(
		f reflect.Kind,
		t reflect.Kind,
		data any,
	) (any, error) {
		return int(42), nil
	}

	f2 := func(
		f reflect.Kind,
		t reflect.Kind,
		data any,
	) (any, error) {
		f2From = f
		return data, nil
	}

	f := ComposeDecodeHookFunc(f1, f2)

	_, err := DecodeHookExec(
		f, reflect.ValueOf(""), reflect.ValueOf([]byte("")))
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if f2From != reflect.Int {
		t.Fatalf("bad: %#v", f2From)
	}
}

func TestOrComposeDecodeHookFunc(t *testing.T) {
	f1 := func(
		f reflect.Kind,
		t reflect.Kind,
		data any,
	) (any, error) {
		return data.(string) + "foo", nil
	}

	f2 := func(
		f reflect.Kind,
		t reflect.Kind,
		data any,
	) (any, error) {
		return data.(string) + "bar", nil
	}

	f := OrComposeDecodeHookFunc(f1, f2)

	result, err := DecodeHookExec(
		f, reflect.ValueOf(""), reflect.ValueOf([]byte("")))
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if result.(string) != "foo" {
		t.Fatalf("bad: %#v", result)
	}
}

func TestOrComposeDecodeHookFunc_correctValueIsLast(t *testing.T) {
	f1 := func(
		f reflect.Kind,
		t reflect.Kind,
		data any,
	) (any, error) {
		return nil, errors.New("f1 error")
	}

	f2 := func(
		f reflect.Kind,
		t reflect.Kind,
		data any,
	) (any, error) {
		return nil, errors.New("f2 error")
	}

	f3 := func(
		f reflect.Kind,
		t reflect.Kind,
		data any,
	) (any, error) {
		return data.(string) + "bar", nil
	}

	f := OrComposeDecodeHookFunc(f1, f2, f3)

	result, err := DecodeHookExec(
		f, reflect.ValueOf(""), reflect.ValueOf([]byte("")))
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if result.(string) != "bar" {
		t.Fatalf("bad: %#v", result)
	}
}

func TestOrComposeDecodeHookFunc_err(t *testing.T) {
	f1 := func(
		f reflect.Kind,
		t reflect.Kind,
		data any,
	) (any, error) {
		return nil, errors.New("f1 error")
	}

	f2 := func(
		f reflect.Kind,
		t reflect.Kind,
		data any,
	) (any, error) {
		return nil, errors.New("f2 error")
	}

	f := OrComposeDecodeHookFunc(f1, f2)

	_, err := DecodeHookExec(
		f, reflect.ValueOf(""), reflect.ValueOf([]byte("")))
	if err == nil {
		t.Fatalf("bad: should return an error")
	}
	if err.Error() != "f1 error\nf2 error\n" {
		t.Fatalf("bad: %s", err)
	}
}

func TestComposeDecodeHookFunc_safe_nofuncs(t *testing.T) {
	f := ComposeDecodeHookFunc()
	type myStruct2 struct {
		MyInt int
	}

	type myStruct1 struct {
		Blah map[string]myStruct2
	}

	src := &myStruct1{Blah: map[string]myStruct2{
		"test": {
			MyInt: 1,
		},
	}}

	dst := &myStruct1{}
	dConf := &DecoderConfig{
		Result:      dst,
		ErrorUnused: true,
		DecodeHook:  f,
	}
	d, err := NewDecoder(dConf)
	if err != nil {
		t.Fatal(err)
	}
	err = d.Decode(src)
	if err != nil {
		t.Fatal(err)
	}
}

func TestComposeDecodeHookFunc_ReflectValueHook(t *testing.T) {
	reflectValueHook := func(
		f reflect.Kind,
		t reflect.Kind,
		data any,
	) (any, error) {
		new := data.(string) + "foo"
		return reflect.ValueOf(new), nil
	}

	stringHook := func(
		f reflect.Kind,
		t reflect.Kind,
		data any,
	) (any, error) {
		return data.(string) + "bar", nil
	}

	f := ComposeDecodeHookFunc(reflectValueHook, stringHook)

	result, err := DecodeHookExec(
		f, reflect.ValueOf(""), reflect.ValueOf([]byte("")))
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if result.(string) != "foobar" {
		t.Fatalf("bad: %#v", result)
	}
}

func TestStringToSliceHookFunc(t *testing.T) {
	f := StringToSliceHookFunc(",")

	strValue := reflect.ValueOf("42")
	sliceValue := reflect.ValueOf([]string{"42"})
	cases := []struct {
		f, t   reflect.Value
		result any
		err    bool
	}{
		{sliceValue, sliceValue, []string{"42"}, false},
		{reflect.ValueOf([]byte("42")), reflect.ValueOf([]byte{}), []byte("42"), false},
		{strValue, strValue, "42", false},
		{
			reflect.ValueOf("foo,bar,baz"),
			sliceValue,
			[]string{"foo", "bar", "baz"},
			false,
		},
		{
			reflect.ValueOf(""),
			sliceValue,
			[]string{},
			false,
		},
	}

	for i, tc := range cases {
		actual, err := DecodeHookExec(f, tc.f, tc.t)
		if tc.err != (err != nil) {
			t.Fatalf("case %d: expected err %#v", i, tc.err)
		}
		if !reflect.DeepEqual(actual, tc.result) {
			t.Fatalf(
				"case %d: expected %#v, got %#v",
				i, tc.result, actual)
		}
	}
}

func TestStringToTimeDurationHookFunc(t *testing.T) {
	f := StringToTimeDurationHookFunc()

	timeValue := reflect.ValueOf(time.Duration(5))
	strValue := reflect.ValueOf("")
	cases := []struct {
		f, t   reflect.Value
		result any
		err    bool
	}{
		{reflect.ValueOf("5s"), timeValue, 5 * time.Second, false},
		{reflect.ValueOf("5"), timeValue, time.Duration(0), true},
		{reflect.ValueOf("5"), strValue, "5", false},
	}

	for i, tc := range cases {
		actual, err := DecodeHookExec(f, tc.f, tc.t)
		if tc.err != (err != nil) {
			t.Fatalf("case %d: expected err %#v", i, tc.err)
		}
		if !reflect.DeepEqual(actual, tc.result) {
			t.Fatalf(
				"case %d: expected %#v, got %#v",
				i, tc.result, actual)
		}
	}
}

func TestStringToURLHookFunc(t *testing.T) {
	f := StringToURLHookFunc()

	urlSample, _ := url.Parse("http://example.com")
	urlValue := reflect.ValueOf(urlSample)
	strValue := reflect.ValueOf("http://example.com")
	cases := []struct {
		f, t   reflect.Value
		result any
		err    bool
	}{
		{reflect.ValueOf("http://example.com"), urlValue, urlSample, false},
		{reflect.ValueOf("http ://example.com"), urlValue, (*url.URL)(nil), true},
		{reflect.ValueOf("http://example.com"), strValue, "http://example.com", false},
	}

	for i, tc := range cases {
		actual, err := DecodeHookExec(f, tc.f, tc.t)
		if tc.err != (err != nil) {
			t.Fatalf("case %d: expected err %#v", i, tc.err)
		}
		if !reflect.DeepEqual(actual, tc.result) {
			t.Fatalf(
				"case %d: expected %#v, got %#v",
				i, tc.result, actual)
		}
	}
}

func TestStringToTimeHookFunc(t *testing.T) {
	strValue := reflect.ValueOf("5")
	timeValue := reflect.ValueOf(time.Time{})
	cases := []struct {
		f, t   reflect.Value
		layout string
		result any
		err    bool
	}{
		{
			reflect.ValueOf("2006-01-02T15:04:05Z"), timeValue, time.RFC3339,
			time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC), false,
		},
		{strValue, timeValue, time.RFC3339, time.Time{}, true},
		{strValue, strValue, time.RFC3339, "5", false},
	}

	for i, tc := range cases {
		f := StringToTimeHookFunc(tc.layout)
		actual, err := DecodeHookExec(f, tc.f, tc.t)
		if tc.err != (err != nil) {
			t.Fatalf("case %d: expected err %#v", i, tc.err)
		}
		if !reflect.DeepEqual(actual, tc.result) {
			t.Fatalf(
				"case %d: expected %#v, got %#v",
				i, tc.result, actual)
		}
	}
}

func TestStringToIPHookFunc(t *testing.T) {
	strValue := reflect.ValueOf("5")
	ipValue := reflect.ValueOf(net.IP{})
	cases := []struct {
		f, t   reflect.Value
		result any
		err    bool
	}{
		{
			reflect.ValueOf("1.2.3.4"), ipValue,
			net.IPv4(0x01, 0x02, 0x03, 0x04), false,
		},
		{strValue, ipValue, net.IP{}, true},
		{strValue, strValue, "5", false},
	}

	for i, tc := range cases {
		f := StringToIPHookFunc()
		actual, err := DecodeHookExec(f, tc.f, tc.t)
		if tc.err != (err != nil) {
			t.Fatalf("case %d: expected err %#v", i, tc.err)
		}
		if !reflect.DeepEqual(actual, tc.result) {
			t.Fatalf(
				"case %d: expected %#v, got %#v",
				i, tc.result, actual)
		}
	}
}

func TestStringToIPNetHookFunc(t *testing.T) {
	strValue := reflect.ValueOf("5")
	ipNetValue := reflect.ValueOf(net.IPNet{})
	var nilNet *net.IPNet = nil

	cases := []struct {
		f, t   reflect.Value
		result any
		err    bool
	}{
		{
			reflect.ValueOf("1.2.3.4/24"), ipNetValue,
			&net.IPNet{
				IP:   net.IP{0x01, 0x02, 0x03, 0x00},
				Mask: net.IPv4Mask(0xff, 0xff, 0xff, 0x00),
			}, false,
		},
		{strValue, ipNetValue, nilNet, true},
		{strValue, strValue, "5", false},
	}

	for i, tc := range cases {
		f := StringToIPNetHookFunc()
		actual, err := DecodeHookExec(f, tc.f, tc.t)
		if tc.err != (err != nil) {
			t.Fatalf("case %d: expected err %#v", i, tc.err)
		}
		if !reflect.DeepEqual(actual, tc.result) {
			t.Fatalf(
				"case %d: expected %#v, got %#v",
				i, tc.result, actual)
		}
	}
}

func TestWeaklyTypedHook(t *testing.T) {
	var f DecodeHookFunc = WeaklyTypedHook

	strValue := reflect.ValueOf("")
	cases := []struct {
		f, t   reflect.Value
		result any
		err    bool
	}{
		// TO STRING
		{
			reflect.ValueOf(false),
			strValue,
			"0",
			false,
		},

		{
			reflect.ValueOf(true),
			strValue,
			"1",
			false,
		},

		{
			reflect.ValueOf(float32(7)),
			strValue,
			"7",
			false,
		},

		{
			reflect.ValueOf(int(7)),
			strValue,
			"7",
			false,
		},

		{
			reflect.ValueOf([]uint8("foo")),
			strValue,
			"foo",
			false,
		},

		{
			reflect.ValueOf(uint(7)),
			strValue,
			"7",
			false,
		},
	}

	for i, tc := range cases {
		actual, err := DecodeHookExec(f, tc.f, tc.t)
		if tc.err != (err != nil) {
			t.Fatalf("case %d: expected err %#v", i, tc.err)
		}
		if !reflect.DeepEqual(actual, tc.result) {
			t.Fatalf(
				"case %d: expected %#v, got %#v",
				i, tc.result, actual)
		}
	}
}

func TestStructToMapHookFuncTabled(t *testing.T) {
	var f DecodeHookFunc = RecursiveStructToMapHookFunc()

	type b struct {
		TestKey string
	}

	type a struct {
		Sub b
	}

	testStruct := a{
		Sub: b{
			TestKey: "testval",
		},
	}

	testMap := map[string]any{
		"Sub": map[string]any{
			"TestKey": "testval",
		},
	}

	cases := []struct {
		name     string
		receiver any
		input    any
		expected any
		err      bool
	}{
		{
			"map receiver",
			func() any {
				var res map[string]any
				return &res
			}(),
			testStruct,
			&testMap,
			false,
		},
		{
			"interface receiver",
			func() any {
				var res any
				return &res
			}(),
			testStruct,
			func() any {
				var exp any = testMap
				return &exp
			}(),
			false,
		},
		{
			"slice receiver errors",
			func() any {
				var res []string
				return &res
			}(),
			testStruct,
			new([]string),
			true,
		},
		{
			"slice to slice - no change",
			func() any {
				var res []string
				return &res
			}(),
			[]string{"a", "b"},
			&[]string{"a", "b"},
			false,
		},
		{
			"string to string - no change",
			func() any {
				var res string
				return &res
			}(),
			"test",
			func() *string {
				s := "test"
				return &s
			}(),
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &DecoderConfig{
				DecodeHook: f,
				Result:     tc.receiver,
			}

			d, err := NewDecoder(cfg)
			if err != nil {
				t.Fatalf("unexpected err %#v", err)
			}

			err = d.Decode(tc.input)
			if tc.err != (err != nil) {
				t.Fatalf("expected err %#v", err)
			}

			if !reflect.DeepEqual(tc.expected, tc.receiver) {
				t.Fatalf("expected %#v, got %#v",
					tc.expected, tc.receiver)
			}
		})
	}
}

func TestTextUnmarshallerHookFunc(t *testing.T) {
	type MyString string

	cases := []struct {
		f, t   reflect.Value
		result any
		err    bool
	}{
		{reflect.ValueOf("42"), reflect.ValueOf(big.Int{}), big.NewInt(42), false},
		{reflect.ValueOf("invalid"), reflect.ValueOf(big.Int{}), nil, true},
		{reflect.ValueOf("5"), reflect.ValueOf("5"), "5", false},
		{reflect.ValueOf(json.Number("42")), reflect.ValueOf(big.Int{}), big.NewInt(42), false},
		{reflect.ValueOf(MyString("42")), reflect.ValueOf(big.Int{}), big.NewInt(42), false},
	}
	for i, tc := range cases {
		f := TextUnmarshallerHookFunc()
		actual, err := DecodeHookExec(f, tc.f, tc.t)
		if tc.err != (err != nil) {
			t.Fatalf("case %d: expected err %#v", i, tc.err)
		}
		if !reflect.DeepEqual(actual, tc.result) {
			t.Fatalf(
				"case %d: expected %#v, got %#v",
				i, tc.result, actual)
		}
	}
}

func TestStringToNetIPAddrHookFunc(t *testing.T) {
	strValue := reflect.ValueOf("5")
	addrValue := reflect.ValueOf(netip.Addr{})
	cases := []struct {
		f, t   reflect.Value
		result any
		err    bool
	}{
		{
			reflect.ValueOf("192.0.2.1"), addrValue,
			netip.AddrFrom4([4]byte{0xc0, 0x00, 0x02, 0x01}), false,
		},
		{strValue, addrValue, netip.Addr{}, true},
		{strValue, strValue, "5", false},
	}

	for i, tc := range cases {
		f := StringToNetIPAddrHookFunc()
		actual, err := DecodeHookExec(f, tc.f, tc.t)
		if tc.err != (err != nil) {
			t.Fatalf("case %d: expected err %#v", i, tc.err)
		}
		if !reflect.DeepEqual(actual, tc.result) {
			t.Fatalf(
				"case %d: expected %#v, got %#v",
				i, tc.result, actual)
		}
	}
}

func TestStringToNetIPAddrPortHookFunc(t *testing.T) {
	strValue := reflect.ValueOf("5")
	addrPortValue := reflect.ValueOf(netip.AddrPort{})
	cases := []struct {
		f, t   reflect.Value
		result any
		err    bool
	}{
		{
			reflect.ValueOf("192.0.2.1:80"), addrPortValue,
			netip.AddrPortFrom(netip.AddrFrom4([4]byte{0xc0, 0x00, 0x02, 0x01}), 80), false,
		},
		{strValue, addrPortValue, netip.AddrPort{}, true},
		{strValue, strValue, "5", false},
	}

	for i, tc := range cases {
		f := StringToNetIPAddrPortHookFunc()
		actual, err := DecodeHookExec(f, tc.f, tc.t)
		if tc.err != (err != nil) {
			t.Fatalf("case %d: expected err %#v", i, tc.err)
		}
		if !reflect.DeepEqual(actual, tc.result) {
			t.Fatalf(
				"case %d: expected %#v, got %#v",
				i, tc.result, actual)
		}
	}
}

func TestStringToNetIPPrefixHookFunc(t *testing.T) {
	strValue := reflect.ValueOf("5")
	prefixValue := reflect.ValueOf(netip.Prefix{})
	cases := []struct {
		f, t   reflect.Value
		result any
		err    bool
	}{
		{
			reflect.ValueOf("192.0.2.1/24"), prefixValue,
			netip.PrefixFrom(netip.AddrFrom4([4]byte{0xc0, 0x00, 0x02, 0x01}), 24),
			false,
		},
		{
			reflect.ValueOf("fd7a:115c::626b:430b/118"), prefixValue,
			netip.PrefixFrom(netip.AddrFrom16([16]byte{0xfd, 0x7a, 0x11, 0x5c, 12: 0x62, 0x6b, 0x43, 0x0b}), 118),
			false,
		},
		{strValue, prefixValue, netip.Prefix{}, true},
		{strValue, strValue, "5", false},
	}

	for i, tc := range cases {
		f := StringToNetIPPrefixHookFunc()
		actual, err := DecodeHookExec(f, tc.f, tc.t)
		if tc.err != (err != nil) {
			t.Fatalf("case %d: expected err %#v", i, tc.err)
		}
		if !reflect.DeepEqual(actual, tc.result) {
			t.Fatalf(
				"case %d:\nexpected %#v,\ngot      %#v",
				i, tc.result, actual)
		}
	}
}

func TestStringToBasicTypeHookFunc(t *testing.T) {
	strValue := reflect.ValueOf("42")

	cases := []struct {
		f, t   reflect.Value
		result any
		err    bool
	}{
		{strValue, strValue, "42", false},
		{strValue, reflect.ValueOf(int8(0)), int8(42), false},
		{strValue, reflect.ValueOf(uint8(0)), uint8(42), false},
		{strValue, reflect.ValueOf(int16(0)), int16(42), false},
		{strValue, reflect.ValueOf(uint16(0)), uint16(42), false},
		{strValue, reflect.ValueOf(int32(0)), int32(42), false},
		{strValue, reflect.ValueOf(uint32(0)), uint32(42), false},
		{strValue, reflect.ValueOf(int64(0)), int64(42), false},
		{strValue, reflect.ValueOf(uint64(0)), uint64(42), false},
		{strValue, reflect.ValueOf(int(0)), int(42), false},
		{strValue, reflect.ValueOf(uint(0)), uint(42), false},
		{strValue, reflect.ValueOf(float32(0)), float32(42), false},
		{strValue, reflect.ValueOf(float64(0)), float64(42), false},
		{reflect.ValueOf("true"), reflect.ValueOf(bool(false)), true, false},
		{strValue, reflect.ValueOf(byte(0)), byte(42), false},
		{strValue, reflect.ValueOf(rune(0)), rune(42), false},
		{strValue, reflect.ValueOf(complex64(0)), complex64(42), false},
		{strValue, reflect.ValueOf(complex128(0)), complex128(42), false},
	}

	for i, tc := range cases {
		f := StringToBasicTypeHookFunc()
		actual, err := DecodeHookExec(f, tc.f, tc.t)
		if tc.err != (err != nil) {
			t.Fatalf("case %d: expected err %#v", i, tc.err)
		}
		if !tc.err && !reflect.DeepEqual(actual, tc.result) {
			t.Fatalf(
				"case %d: expected %#v, got %#v",
				i, tc.result, actual)
		}
	}
}

func TestStringToInt8HookFunc(t *testing.T) {
	suite := decodeHookTestSuite[string, int8]{
		fn: StringToInt8HookFunc(),
		ok: []decodeHookTestCase[string, int8]{
			{"42", 42},
			{"-42", int8(-42)},
			{"0b101010", int8(42)},
			{"052", int8(42)},
			{"0o52", int8(42)},
			{"0x2a", int8(42)},
			{"0X2A", int8(42)},
			{"0", int8(0)},
		},
		fail: []decodeHookFailureTestCase[string, int8]{
			{strings.Repeat("42", 42)},
			{"42.42"},
			{"0.0"},
		},
	}

	suite.Run(t)
}

func TestStringToUint8HookFunc(t *testing.T) {
	suite := decodeHookTestSuite[string, uint8]{
		fn: StringToUint8HookFunc(),
		ok: []decodeHookTestCase[string, uint8]{
			{"42", 42},
			{"0b101010", uint8(42)},
			{"052", uint8(42)},
			{"0o52", uint8(42)},
			{"0x2a", uint8(42)},
			{"0X2A", uint8(42)},
			{"0", uint8(0)},
		},
		fail: []decodeHookFailureTestCase[string, uint8]{
			{strings.Repeat("42", 42)},
			{"42.42"},
			{"-42"},
			{"0.0"},
		},
	}

	suite.Run(t)
}

func TestStringToInt16HookFunc(t *testing.T) {
	suite := decodeHookTestSuite[string, int16]{
		fn: StringToInt16HookFunc(),
		ok: []decodeHookTestCase[string, int16]{
			{"42", 42},
			{"-42", int16(-42)},
			{"0b101010", int16(42)},
			{"052", int16(42)},
			{"0o52", int16(42)},
			{"0x2a", int16(42)},
			{"0X2A", int16(42)},
			{"0", int16(0)},
		},
		fail: []decodeHookFailureTestCase[string, int16]{
			{strings.Repeat("42", 42)},
			{"42.42"},
			{"0.0"},
		},
	}

	suite.Run(t)
}

func TestStringToUint16HookFunc(t *testing.T) {
	suite := decodeHookTestSuite[string, uint16]{
		fn: StringToUint16HookFunc(),
		ok: []decodeHookTestCase[string, uint16]{
			{"42", 42},
			{"0b101010", uint16(42)},
			{"052", uint16(42)},
			{"0o52", uint16(42)},
			{"0x2a", uint16(42)},
			{"0X2A", uint16(42)},
			{"0", uint16(0)},
		},
		fail: []decodeHookFailureTestCase[string, uint16]{
			{strings.Repeat("42", 42)},
			{"42.42"},
			{"-42"},
			{"0.0"},
		},
	}

	suite.Run(t)
}

func TestStringToInt32HookFunc(t *testing.T) {
	suite := decodeHookTestSuite[string, int32]{
		fn: StringToInt32HookFunc(),
		ok: []decodeHookTestCase[string, int32]{
			{"42", 42},
			{"-42", int32(-42)},
			{"0b101010", int32(42)},
			{"052", int32(42)},
			{"0o52", int32(42)},
			{"0x2a", int32(42)},
			{"0X2A", int32(42)},
			{"0", int32(0)},
		},
		fail: []decodeHookFailureTestCase[string, int32]{
			{strings.Repeat("42", 42)},
			{"42.42"},
			{"0.0"},
		},
	}

	suite.Run(t)
}

func TestStringToUint32HookFunc(t *testing.T) {
	suite := decodeHookTestSuite[string, uint32]{
		fn: StringToUint32HookFunc(),
		ok: []decodeHookTestCase[string, uint32]{
			{"42", 42},
			{"0b101010", uint32(42)},
			{"052", uint32(42)},
			{"0o52", uint32(42)},
			{"0x2a", uint32(42)},
			{"0X2A", uint32(42)},
			{"0", uint32(0)},
		},
		fail: []decodeHookFailureTestCase[string, uint32]{
			{strings.Repeat("42", 42)},
			{"42.42"},
			{"-42"},
			{"0.0"},
		},
	}

	suite.Run(t)
}

func TestStringToInt64HookFunc(t *testing.T) {
	suite := decodeHookTestSuite[string, int64]{
		fn: StringToInt64HookFunc(),
		ok: []decodeHookTestCase[string, int64]{
			{"42", 42},
			{"-42", int64(-42)},
			{"0b101010", int64(42)},
			{"052", int64(42)},
			{"0o52", int64(42)},
			{"0x2a", int64(42)},
			{"0X2A", int64(42)},
			{"0", int64(0)},
		},
		fail: []decodeHookFailureTestCase[string, int64]{
			{strings.Repeat("42", 42)},
			{"42.42"},
			{"0.0"},
		},
	}

	suite.Run(t)
}

func TestStringToUint64HookFunc(t *testing.T) {
	suite := decodeHookTestSuite[string, uint64]{
		fn: StringToUint64HookFunc(),
		ok: []decodeHookTestCase[string, uint64]{
			{"42", 42},
			{"0b101010", uint64(42)},
			{"052", uint64(42)},
			{"0o52", uint64(42)},
			{"0x2a", uint64(42)},
			{"0X2A", uint64(42)},
			{"0", uint64(0)},
		},
		fail: []decodeHookFailureTestCase[string, uint64]{
			{strings.Repeat("42", 42)},
			{"42.42"},
			{"-42"},
			{"0.0"},
		},
	}

	suite.Run(t)
}

func TestStringToIntHookFunc(t *testing.T) {
	suite := decodeHookTestSuite[string, int]{
		fn: StringToIntHookFunc(),
		ok: []decodeHookTestCase[string, int]{
			{"42", 42},
			{"-42", int(-42)},
			{"0b101010", int(42)},
			{"052", int(42)},
			{"0o52", int(42)},
			{"0x2a", int(42)},
			{"0X2A", int(42)},
			{"0", int(0)},
		},
		fail: []decodeHookFailureTestCase[string, int]{
			{strings.Repeat("42", 42)},
			{"42.42"},
			{"0.0"},
		},
	}

	suite.Run(t)
}

func TestStringToUintHookFunc(t *testing.T) {
	suite := decodeHookTestSuite[string, uint]{
		fn: StringToUintHookFunc(),
		ok: []decodeHookTestCase[string, uint]{
			{"42", 42},
			{"0b101010", uint(42)},
			{"052", uint(42)},
			{"0o52", uint(42)},
			{"0x2a", uint(42)},
			{"0X2A", uint(42)},
			{"0", uint(0)},
		},
		fail: []decodeHookFailureTestCase[string, uint]{
			{strings.Repeat("42", 42)},
			{"42.42"},
			{"-42"},
			{"0.0"},
		},
	}

	suite.Run(t)
}

func TestStringToFloat32HookFunc(t *testing.T) {
	suite := decodeHookTestSuite[string, float32]{
		fn: StringToFloat32HookFunc(),
		ok: []decodeHookTestCase[string, float32]{
			{"42.42", float32(42.42)},
			{"-42.42", float32(-42.42)},
			{"0", float32(0)},
			{"1e3", float32(1000)},
			{"1e-3", float32(0.001)},
		},
		fail: []decodeHookFailureTestCase[string, float32]{
			{strings.Repeat("42", 420)},
			{"42.42.42"},
		},
	}

	suite.Run(t)
}

func TestStringToFloat64HookFunc(t *testing.T) {
	suite := decodeHookTestSuite[string, float64]{
		fn: StringToFloat64HookFunc(),
		ok: []decodeHookTestCase[string, float64]{
			{"42.42", float64(42.42)},
			{"-42.42", float64(-42.42)},
			{"0", float64(0)},
			{"0.0", float64(0)},
			{"1e3", float64(1000)},
			{"1e-3", float64(0.001)},
		},
		fail: []decodeHookFailureTestCase[string, float64]{
			{strings.Repeat("42", 420)},
			{"42.42.42"},
		},
	}

	suite.Run(t)
}

func TestStringToComplex64HookFunc(t *testing.T) {
	suite := decodeHookTestSuite[string, complex64]{
		fn: StringToComplex64HookFunc(),
		ok: []decodeHookTestCase[string, complex64]{
			{"42.42+42.42i", complex(float32(42.42), float32(42.42))},
			{"-42.42", complex(float32(-42.42), 0)},
			{"0", complex(float32(0), 0)},
			{"0.0", complex(float32(0), 0)},
			{"1e3", complex(float32(1000), 0)},
			{"1e-3", complex(float32(0.001), 0)},
			{"1e3i", complex(float32(0), 1000)},
			{"1e-3i", complex(float32(0), 0.001)},
		},
		fail: []decodeHookFailureTestCase[string, complex64]{
			{strings.Repeat("42", 420)},
			{"42.42.42"},
		},
	}

	suite.Run(t)
}

func TestStringToComplex128HookFunc(t *testing.T) {
	suite := decodeHookTestSuite[string, complex128]{
		fn: StringToComplex128HookFunc(),
		ok: []decodeHookTestCase[string, complex128]{
			{"42.42+42.42i", complex(42.42, 42.42)},
			{"-42.42", complex(-42.42, 0)},
			{"0", complex(0, 0)},
			{"0.0", complex(0, 0)},
			{"1e3", complex(1000, 0)},
			{"1e-3", complex(0.001, 0)},
			{"1e3i", complex(0, 1000)},
			{"1e-3i", complex(0, 0.001)},
		},
		fail: []decodeHookFailureTestCase[string, complex128]{
			{strings.Repeat("42", 420)},
			{"42.42.42"},
		},
	}

	suite.Run(t)
}

func TestErrorLeakageDecodeHook(t *testing.T) {
	cases := []struct {
		value         any
		target        any
		hook          DecodeHookFunc
		allowNilError bool
	}{
		// case 0
		{1234, []string{}, StringToSliceHookFunc(","), true},
		{"testing", time.Second, StringToTimeDurationHookFunc(), false},
		{":testing", &url.URL{}, StringToURLHookFunc(), false},
		{"testing", net.IP{}, StringToIPHookFunc(), false},
		{"testing", net.IPNet{}, StringToIPNetHookFunc(), false},
		// case 5
		{"testing", time.Time{}, StringToTimeHookFunc(time.RFC3339), false},
		{"testing", time.Time{}, StringToTimeHookFunc(time.RFC3339), false},
		{true, true, WeaklyTypedHook, true},
		{true, "string", WeaklyTypedHook, true},
		{1.0, "string", WeaklyTypedHook, true},
		// case 10
		{1, "string", WeaklyTypedHook, true},
		{[]uint8{0x00}, "string", WeaklyTypedHook, true},
		{uint(0), "string", WeaklyTypedHook, true},
		{struct{}{}, struct{}{}, RecursiveStructToMapHookFunc(), true},
		{"testing", netip.Addr{}, StringToNetIPAddrHookFunc(), false},
		// case 15
		{"testing:testing", netip.AddrPort{}, StringToNetIPAddrPortHookFunc(), false},
		{"testing", netip.Prefix{}, StringToNetIPPrefixHookFunc(), false},
		{"testing", int8(0), StringToInt8HookFunc(), false},
		{"testing", uint8(0), StringToUint8HookFunc(), false},
		// case 20
		{"testing", int16(0), StringToInt16HookFunc(), false},
		{"testing", uint16(0), StringToUint16HookFunc(), false},
		{"testing", int32(0), StringToInt32HookFunc(), false},
		{"testing", uint32(0), StringToUint32HookFunc(), false},
		{"testing", int64(0), StringToInt64HookFunc(), false},
		// case 25
		{"testing", uint64(0), StringToUint64HookFunc(), false},
		{"testing", int(0), StringToIntHookFunc(), false},
		{"testing", uint(0), StringToUintHookFunc(), false},
		{"testing", float32(0), StringToFloat32HookFunc(), false},
		{"testing", float64(0), StringToFloat64HookFunc(), false},
		// case 30
		{"testing", true, StringToBoolHookFunc(), false},
		{"testing", byte(0), StringToByteHookFunc(), false},
		{"testing", rune(0), StringToRuneHookFunc(), false},
		{"testing", complex64(0), StringToComplex64HookFunc(), false},
		{"testing", complex128(0), StringToComplex128HookFunc(), false},
	}

	for i, tc := range cases {
		value := reflect.ValueOf(tc.value)
		target := reflect.ValueOf(tc.target)
		output, err := DecodeHookExec(tc.hook, value, target)

		if err == nil {
			if tc.allowNilError {
				continue
			}

			t.Fatalf("case %d: expected error from input %v:\n\toutput (%T): %#v\n\toutput (string): %v", i, tc.value, output, output, output)
		}

		strValue := fmt.Sprintf("%v", tc.value)
		if strings.Contains(err.Error(), strValue) {
			t.Errorf("case %d: error contains input value\n\terr: %v\n\tinput: %v", i, err, strValue)
		} else {
			t.Logf("case %d: got safe error: %v", i, err)
		}
	}
}
