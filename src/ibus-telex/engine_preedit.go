/*
 * Telex - A Vietnamese Input method editor
 * Copyright (C) 2018 Luong Thanh Lam <ltlam93@gmail.com>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package main

import (
	"log"
	"strings"

	"github.com/andodevel/goibus/ibus"
	"github.com/andodevel/telex-core"
	"github.com/godbus/dbus"
)

func (e *IBusTelex) preeditProcessKeyEvent(keyVal uint32, keyCode uint32, state uint32) (bool, *dbus.Error) {
	var rawKeyLen = e.getRawKeyLen()
	var keyRune = rune(keyVal)
	var oldText = e.getPreeditString()
	defer e.updateLastKeyWithShift(keyVal, state)

	// workaround for chrome's address bar and Google SpreadSheets
	if !e.isValidState(state) || !e.canProcessKey(keyVal) ||
		(rawKeyLen == 0 && !e.preeditor.CanProcessKey(keyRune)) {
		if rawKeyLen > 0 {
			e.HidePreeditText()
			e.commitText(e.getPreeditString())
			e.preeditor.Reset()
		}
		return false, nil
	}

	if keyVal == IBusBackSpace {
		if rawKeyLen > 0 {
			e.preeditor.RemoveLastChar(true)
			e.updatePreedit(e.getPreeditString())
			return true, nil
		}
		return false, nil
	}
	if keyVal == IBusTab {
		e.commitPreedit(e.getComposedString(oldText))
		return false, nil
	}

	if e.preeditor.CanProcessKey(keyRune) {
		if state&IBusLockMask != 0 {
			keyRune = e.toUpper(keyRune)
		}
		e.preeditor.ProcessKey(keyRune, e.getTelexInputMode())
		if inKeyList(e.preeditor.GetInputMethod().AppendingKeys, keyRune) {
			if fullSeq := e.preeditor.GetProcessedString(telex.VietnameseMode); len(fullSeq) > 0 && rune(fullSeq[len(fullSeq)-1]) == keyRune {
				e.commitPreedit(fullSeq)
			} else if newText := e.getPreeditString(); newText != "" && keyRune == rune(newText[len(newText)-1]) {
				e.commitPreedit(oldText + string(keyRune))
			} else {
				e.updatePreedit(e.getPreeditString())
			}
		} else {
			e.updatePreedit(e.getPreeditString())
		}
		return true, nil
	} else if telex.IsWordBreakSymbol(keyRune) {
		e.commitPreedit(e.getComposedString(oldText) + string(keyRune))
		return true, nil
	}
	e.commitPreedit(e.getPreeditString())
	return false, nil
}

func (e *IBusTelex) updatePreedit(processedStr string) {
	var encodedStr = e.encodeText(processedStr)
	var preeditLen = uint32(len([]rune(encodedStr)))
	if preeditLen == 0 {
		e.HidePreeditText()
		e.CommitText(ibus.NewText(""))
		return
	}
	var ibusText = ibus.NewText(encodedStr)
	ibusText.AppendAttr(ibus.IBUS_ATTR_TYPE_NONE, ibus.IBUS_ATTR_UNDERLINE_SINGLE, 0, preeditLen)
	e.UpdatePreeditTextWithMode(ibusText, preeditLen, true, ibus.IBUS_ENGINE_PREEDIT_COMMIT)

	if e.config.IBflags&IBmouseCapturing != 0 {
		mouseCaptureUnlock()
	}
}

func (e *IBusTelex) getWhiteList() [][]string {
	return [][]string{
		e.config.PreeditWhiteList,
		e.config.SurroundingTextWhiteList,
		e.config.ForwardKeyWhiteList,
		e.config.SLForwardKeyWhiteList,
		e.config.X11ClipboardWhiteList,
		e.config.DirectForwardKeyWhiteList,
		e.config.ExceptedList,
	}
}

func (e *IBusTelex) getTelexInputMode() telex.Mode {
	if e.shouldFallbackToEnglish(false) {
		return telex.EnglishMode
	}
	return telex.VietnameseMode
}

func (e *IBusTelex) shouldFallbackToEnglish(checkVnRune bool) bool {
	if e.config.IBflags&IBautoNonVnRestore == 0 {
		return false
	}
	var vnSeq = e.getProcessedString(telex.VietnameseMode | telex.LowerCase)
	var vnRunes = []rune(vnSeq)
	if len(vnRunes) == 0 {
		return false
	}
	// we want to allow dd even in non-vn sequence, because dd is used a lot in abbreviation
	if e.config.IBflags&IBddFreeStyle != 0 && (vnRunes[len(vnRunes)-1] == 'd' || strings.ContainsRune(vnSeq, 'đ')) {
		return false
	}
	if checkVnRune && !telex.HasAnyVietnameseRune(vnSeq) {
		return false
	}
	return !e.preeditor.IsValid(false)
}

func (e *IBusTelex) mustFallbackToEnglish() bool {
	if e.config.IBflags&IBautoNonVnRestore == 0 {
		return false
	}
	var vnSeq = e.getProcessedString(telex.VietnameseMode | telex.LowerCase)
	var vnRunes = []rune(vnSeq)
	if len(vnRunes) == 0 {
		return false
	}
	return !e.preeditor.IsValid(true)
}

func (e *IBusTelex) getComposedString(oldText string) string {
	if telex.HasAnyVietnameseRune(oldText) && e.mustFallbackToEnglish() {
		return e.getProcessedString(telex.EnglishMode)
	}
	return oldText
}

func (e *IBusTelex) encodeText(text string) string {
	return telex.Encode(e.config.OutputCharset, text)
}

func (e *IBusTelex) getProcessedString(mode telex.Mode) string {
	return e.preeditor.GetProcessedString(mode)
}

func (e *IBusTelex) getPreeditString() string {
	if e.shouldFallbackToEnglish(true) {
		return e.getProcessedString(telex.EnglishMode)
	}
	return e.getProcessedString(telex.VietnameseMode)
}

func (e *IBusTelex) resetPreedit() {
	e.HidePreeditText()
	e.preeditor.Reset()
}

func (e *IBusTelex) commitPreedit(s string) {
	e.commitText(s)
	e.HidePreeditText()
	e.preeditor.Reset()
}

func (e *IBusTelex) commitText(str string) {
	if str == "" {
		return
	}
	log.Printf("Commit Text [%s]\n", str)
	e.CommitText(ibus.NewText(e.encodeText(str)))
}

func (e *IBusTelex) getVnSeq() string {
	return e.preeditor.GetProcessedString(telex.VietnameseMode)
}
