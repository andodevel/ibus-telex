/*
 * Telex - A Vietnamese Input method editor
 * Copyright (C) Luong Thanh Lam <ltlam93@gmail.com>
 *
 * This software is licensed under the MIT license. For more information,
 * see <https://github.com/andodevel/telex-core/blob/master/LICENSE>.
 */

package telex

const UNICODE = "Unicode"

func Encode(charsetName string, input string) string {
	return input
}

func GetCharsetNames() []string {
	return []string{UNICODE}
}
