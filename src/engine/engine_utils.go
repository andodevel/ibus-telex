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
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/andodevel/ibus-telex/src/core"
	"github.com/andodevel/ibus-telex/src/x11"
	"github.com/BambooEngine/goibus/ibus"
	"github.com/godbus/dbus"
)

func GetIBusEngineCreator() func(*dbus.Conn, string) dbus.ObjectPath {
	go keyPressCapturing()

	return func(conn *dbus.Conn, ngName string) dbus.ObjectPath {
		var engineName = strings.ToLower(ngName)
		var engine = new(IBusTelex)
		var config = loadConfig(engineName)
		var objectPath = dbus.ObjectPath(fmt.Sprintf("/org/freedesktop/IBus/Engine/%s/%d", engineName, time.Now().UnixNano()))
		var inputMethod = core.ParseInputMethod(config.InputMethodDefinitions, config.InputMethod)
		engine.Engine = ibus.BaseEngine(conn, objectPath)
		engine.engineName = engineName
		engine.preeditor = core.NewEngine(inputMethod, config.Flags)
		engine.config = loadConfig(engineName)
		engine.propList = GetPropListByConfig(config)
		ibus.PublishEngine(conn, objectPath, engine)
		go engine.init()

		return objectPath
	}
}

const KeypressDelayMs = 10

func (e *IBusTelex) init() {
	keyPressHandler = e.keyPressHandler

	if e.config.IBflags&IBmouseCapturing != 0 {
		x11.StartMouseCapturing()
	}
	x11.StartMouseRecording()
	x11.OnMouseMove = func() {
		e.Lock()
		defer e.Unlock()
		if e.checkInputMode(preeditIM) {
			if e.getRawKeyLen() == 0 {
				return
			}
			e.commitPreedit(e.getPreeditString())
		}
	}
	x11.OnMouseClick = func() {
		e.Lock()
		defer e.Unlock()

		e.resetFakeBackspace()
		e.resetBuffer()
		e.keyPressDelay = KeypressDelayMs
		if e.capabilities&IBusCapSurroundingText != 0 {
			//e.ForwardKeyEvent(IBUS_Shift_R, XK_Shift_R-8, 0)
			x11.SendShiftR()
			e.isSurroundingTextReady = true
			e.keyPressDelay = KeypressDelayMs * 10
		}
	}
}

var keyPressHandler = func(keyVal, keyCode, state uint32) {}
var keyPressChan = make(chan [3]uint32, 100)

func keyPressCapturing() {
	for keyEvents := range keyPressChan {
		var keyVal, keyCode, state = keyEvents[0], keyEvents[1], keyEvents[2]
		keyPressHandler(keyVal, keyCode, state)
	}
}

func (e *IBusTelex) resetBuffer() {
	if e.getRawKeyLen() == 0 {
		return
	}
	if e.checkInputMode(preeditIM) {
		e.commitPreedit(e.getPreeditString())
	} else {
		e.preeditor.Reset()
	}
}

func (e *IBusTelex) toUpper(keyRune rune) rune {
	var keyMapping = map[rune]rune{
		'[': '{',
		']': '}',
		'{': '[',
		'}': ']',
	}

	if upperSpecialKey, found := keyMapping[keyRune]; found && inKeyList(e.preeditor.GetInputMethod().AppendingKeys, keyRune) {
		keyRune = upperSpecialKey
	}
	return keyRune
}

func (e *IBusTelex) updateLastKeyWithShift(keyVal, state uint32) {
	if e.canProcessKey(keyVal) {
		e.lastKeyWithShift = state&IBusShiftMask != 0
	} else {
		e.lastKeyWithShift = false
	}
}

func (e *IBusTelex) isIgnoredKey(keyVal, state uint32) bool {
	if state&IBusReleaseMask != 0 {
		//Ignore key-up event
		return true
	}
	if keyVal == IBusCapsLock {
		return true
	}
	if e.checkInputMode(usIM) {
		if e.isInputModeLTOpened || keyVal == IBusOpenLookupTable {
			return false
		}
		return true
	}
	return false
}

func (e *IBusTelex) getRawKeyLen() int {
	return len(e.preeditor.GetProcessedString(core.EnglishMode | core.FullText))
}

func (e *IBusTelex) getInputMode() int {
	if e.wmClasses != "" {
		if im, ok := e.config.InputModeMapping[e.wmClasses]; ok && imLookupTable[im] != "" {
			return im
		}
	}
	if imLookupTable[e.config.DefaultInputMode] != "" {
		return e.config.DefaultInputMode
	}
	return preeditIM
}

func (e *IBusTelex) ltProcessKeyEvent(keyVal uint32, keyCode uint32, state uint32) (bool, *dbus.Error) {
	var wmClasses = x11.GetFocusWindowClass()
	//e.HideLookupTable()
	fmt.Printf("keyCode 0x%04x keyval 0x%04x | %c\n", keyCode, keyVal, rune(keyVal))
	//e.HideAuxiliaryText()
	if wmClasses == "" {
		return true, nil
	}
	if keyVal == IBusOpenLookupTable {
		e.closeInputModeCandidates()
		return false, nil
	}
	var keyRune = rune(keyVal)
	if keyVal == IBusLeft || keyVal == IBusUp {
		e.CursorUp()
		return true, nil
	} else if keyVal == IBusRight || keyVal == IBusDown {
		e.CursorDown()
		return true, nil
	} else if keyVal == IBusPageUp {
		e.PageUp()
		return true, nil
	} else if keyVal == IBusPageDown {
		e.PageDown()
		return true, nil
	}
	if keyVal == IBusReturn {
		e.commitInputModeCandidate()
		e.closeInputModeCandidates()
		return true, nil
	}
	if keyRune >= '1' && keyRune <= '7' {
		if pos, err := strconv.Atoi(string(keyRune)); err == nil {
			if e.inputModeLookupTable.SetCursorPos(uint32(pos - 1)) {
				e.commitInputModeCandidate()
				e.closeInputModeCandidates()
				return true, nil
			} else {
				e.closeInputModeCandidates()
			}
		}
	}
	e.closeInputModeCandidates()
	return false, nil
}

func (e *IBusTelex) commitInputModeCandidate() {
	var im = e.inputModeLookupTable.CursorPos + 1
	e.config.InputModeMapping[e.wmClasses] = int(im)

	saveConfig(e.config, e.engineName)
	e.propList = GetPropListByConfig(e.config)
	e.RegisterProperties(e.propList)
}

func (e *IBusTelex) closeInputModeCandidates() {
	e.inputModeLookupTable = nil
	e.UpdateLookupTable(ibus.NewLookupTable(), true) // workaround for issue #18
	e.HidePreeditText()
	e.HideLookupTable()
	e.HideAuxiliaryText()
	e.isInputModeLTOpened = false
}

func (e *IBusTelex) updateInputModeLT() {
	var visible = len(e.inputModeLookupTable.Candidates) > 0
	e.UpdateLookupTable(e.inputModeLookupTable, visible)
}

func (e *IBusTelex) isValidState(state uint32) bool {
	if state&IBusControlMask != 0 ||
		state&IBusMod1Mask != 0 ||
		state&IBusIgnoredMask != 0 ||
		state&IBusSuperMask != 0 ||
		state&IBusHyperMask != 0 ||
		state&IBusMetaMask != 0 {
		return false
	}
	return true
}

func (e *IBusTelex) canProcessKey(keyVal uint32) bool {
	var keyRune = rune(keyVal)
	if keyVal == IBusSpace || keyVal == IBusBackSpace || core.IsWordBreakSymbol(keyRune) {
		return true
	}
	return e.preeditor.CanProcessKey(keyRune)
}

func (e *IBusTelex) inBackspaceWhiteList() bool {
	var inputMode = e.getInputMode()
	for _, im := range imBackspaceList {
		if im == inputMode {
			return true
		}
	}
	return false
}

func (e *IBusTelex) inBrowserList() bool {
	return inStringList(DefaultBrowserList, e.wmClasses)
}

func (e *IBusTelex) checkInputMode(im int) bool {
	return e.getInputMode() == im
}

func notify(enMode bool) {
	var title = "Vietnamese"
	var msg = "Press Shift to switch to English"
	if enMode {
		title = "English"
		msg = "Press Shift to switch to Vietnamese"
	}
	conn, err := dbus.SessionBus()
	if err != nil {
		fmt.Println(err)
		return
	}
	obj := conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications")
	call := obj.Call("org.freedesktop.Notifications.Notify", 0, "", uint32(281025),
		"", title, msg, []string{}, map[string]dbus.Variant{}, int32(3000))
	if call.Err != nil {
		fmt.Println(call.Err)
	}
}
