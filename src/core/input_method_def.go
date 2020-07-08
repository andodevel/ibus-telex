/*
 * Telex - A Vietnamese Input method editor
 * Copyright (C) Luong Thanh Lam <ltlam93@gmail.com>
 *
 * This software is licensed under the MIT license. For more information,
 * see <https://github.com/andodevel/ibus-telex/src/core/blob/master/LICENSE>.
 */

package telex

type InputMethodDefinition map[string]string

var InputMethodDefinitions = map[string]InputMethodDefinition{
	"Telex": {
		"z": "XoaDauThanh",
		"s": "DauSac",
		"f": "DauHuyen",
		"r": "DauHoi",
		"x": "DauNga",
		"j": "DauNang",
		"a": "A_Â",
		"e": "E_Ê",
		"o": "O_Ô",
		"w": "UOA_ƯƠĂ",
		"d": "D_Đ",
	},
}

func GetInputMethodDefinitions() map[string]InputMethodDefinition {
	var t = make(map[string]InputMethodDefinition)
	for k, v := range InputMethodDefinitions {
		t[k] = v
	}
	return t
}
