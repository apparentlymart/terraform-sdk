// Code generated by "stringer -type=ValueType"; DO NOT EDIT.

package tflegacy

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[TypeInvalid-0]
	_ = x[TypeBool-1]
	_ = x[TypeInt-2]
	_ = x[TypeFloat-3]
	_ = x[TypeString-4]
	_ = x[TypeList-5]
	_ = x[TypeMap-6]
	_ = x[TypeSet-7]
	_ = x[typeObject-8]
}

const _ValueType_name = "TypeInvalidTypeBoolTypeIntTypeFloatTypeStringTypeListTypeMapTypeSettypeObject"

var _ValueType_index = [...]uint8{0, 11, 19, 26, 35, 45, 53, 60, 67, 77}

func (i ValueType) String() string {
	if i < 0 || i >= ValueType(len(_ValueType_index)-1) {
		return "ValueType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _ValueType_name[_ValueType_index[i]:_ValueType_index[i+1]]
}