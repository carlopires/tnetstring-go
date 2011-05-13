package tnetstring

import (
	"os"
	"reflect"
	"strconv"
)

type outbuf struct {
	buf []byte
	// n is the index we last wrote to or the amount of space left,
	// depending on how you look at it
	n int
}

func Marshal(v interface{}) (string, os.Error) {
	val := reflect.ValueOf(v)
	b := new(outbuf)
	b.n = 20
	b.buf = make([]byte, b.n)
	if err := marshal(b, val); err != nil {
		return "", err
	}
	return string(b.buf[b.n:]), nil
}

var typeLookup = [...]byte{
	reflect.Invalid: '~',
	reflect.Bool:    '!',
	reflect.Int:     '#',
	reflect.Int8:    '#',
	reflect.Int16:   '#',
	reflect.Int32:   '#',
	reflect.Int64:   '#',
	reflect.Uint:    '#',
	reflect.Uint8:   '#',
	reflect.Uint16:  '#',
	reflect.Uint32:  '#',
	reflect.Uint64:  '#',
	reflect.Uintptr: '#',
	reflect.String:  ',',
	reflect.Array:   ']',
	reflect.Slice:   ']',
	reflect.Map:     '}',
	reflect.Struct:  '}',
	// include last item so the array has the right length
	reflect.UnsafePointer: 0,
}

func marshal(b *outbuf, v reflect.Value) os.Error {
	v = indirect(v, false)
	kind := v.Kind()
	typ := typeLookup[kind]
	if typ == 0 {
		return os.NewError("tnetstring: unsupported type")
	}
	b.writeByte(typ)
	orig := len(b.buf) - b.n
	switch typ {
	case '~':
	case '!':
		str := "false"
		if v.Bool() {
			str = "true"
		}
		b.writeString(str)
	case '#':
		var str string
		if reflect.Int <= kind && kind <= reflect.Int64 {
			str = strconv.Itoa64(v.Int())
		} else {
			str = strconv.Uitoa64(v.Uint())
		}
		b.writeString(str)
	case ',':
		b.writeString(v.String())
	case ']':
		l := v.Len()
		for i := l - 1; i >= 0; i-- {
			if err := marshal(b, v.Index(i)); err != nil {
				return err
			}
		}
	case '}':
		if kind == reflect.Map {
			if v.Type().Key().Kind() != reflect.String {
				return os.NewError("tnetstring: only maps with string keys can be marshaled")
			}
			for _, key := range v.MapKeys() {
				if err := marshal(b, v.MapIndex(key)); err != nil {
					return err
				}
				b.writeByte(',')
				orig := len(b.buf) - b.n
				b.writeString(key.String())
				b.writeLen(orig)
			}
		} else {
			t := v.Type()
			l := t.NumField()
			for i := l - 1; i >= 0; i-- {
				field := t.Field(i)
				str := field.Tag
				if str == "" {
					str = field.Name
				}
				if err := marshal(b, v.Field(i)); err != nil {
					return err
				}
				b.writeByte(',')
				orig := len(b.buf) - b.n
				b.writeString(str)
				b.writeLen(orig)
			}
		}
	default:
		panic("unreachable")
	}
	b.writeLen(orig)
	return nil
}

func (buf *outbuf) writeByte(b byte) {
	if buf.n <= 0 {
		buf.grow(1)
	}
	buf.n--
	buf.buf[buf.n] = b
}

func (b *outbuf) writeString(s string) {
	if b.n < len(s) {
		b.grow(len(s))
	}
	b.n -= len(s)
	copy(b.buf[b.n:], s)
}

func (b *outbuf) writeLen(orig int) {
	l := len(b.buf) - b.n - orig
	str := strconv.Itoa(l)
	b.writeByte(':')
	b.writeString(str)
}

func (b *outbuf) grow(n int) {
	l := len(b.buf)
	need := 2 * l
	if need < l+n-b.n {
		need = l + n - b.n
	}
	buf := make([]byte, need)
	copy(buf[need-l+b.n:], b.buf[b.n:])
	b.buf = buf
	b.n = need - l + b.n
}
