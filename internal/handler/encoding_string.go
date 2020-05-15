// Code generated by "stringer -type=encoding -linecomment"; DO NOT EDIT.

package handler

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[encIdentity-1]
	_ = x[encGzip-2]
	_ = x[encDeflate-4]
}

const (
	_encoding_name_0 = "identitygzip"
	_encoding_name_1 = "deflate"
)

var (
	_encoding_index_0 = [...]uint8{0, 8, 12}
)

func (i encoding) String() string {
	switch {
	case 1 <= i && i <= 2:
		i -= 1
		return _encoding_name_0[_encoding_index_0[i]:_encoding_index_0[i+1]]
	case i == 4:
		return _encoding_name_1
	default:
		return "encoding(" + strconv.FormatInt(int64(i), 10) + ")"
	}
}
