package util

import (
	"strconv"
	"strings"
)

// URLDecode decodes a string to a plain URL once
func URLDecode(body string) string {
	var i int
	buf := strings.Builder{}
	for i = 0; i < len(body)-2; i++ {
		ch := body[i]
		switch ch {
		case 0x25:
			value, err := strconv.ParseInt(body[i+1:i+3], 16, 64)
			if err != nil {
				buf.WriteByte(body[i])
				buf.WriteByte(body[i+1])
				buf.WriteByte(body[i+2])
			} else {
				buf.WriteByte(byte(value))
			}
			i += 2
		default:
			buf.WriteByte(ch)
		}
	}
	buf.WriteString(body[i:])
	return buf.String()
}
