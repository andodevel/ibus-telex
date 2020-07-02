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
	"log"
	"os/exec"
	"reflect"
	"sync"

	"github.com/andodevel/goibus/ibus"
	"github.com/andodevel/telex-core"
	"github.com/godbus/dbus"
)

type IBusandodevel struct {
	sync.Mutex
	ibus.Engine
	preeditor              telex.IEngine
	engineName             string
	config                 *Config
	propList               *ibus.PropList
	wmClasses              string
	isInputModeLTOpened    bool
	inputModeLookupTable   *ibus.LookupTable
	capabilities           uint32
	keyPressDelay          int
	nFakeBackSpace         int
	isFirstTimeSendingBS   bool
	isSurroundingTextReady bool
	lastKeyWithShift       bool
}

/**
Implement IBus.Engine's process_key_event default signal handler.

Args:
	keyval - The keycode, transformed through a keymap, stays the
		same for every keyboard
	keycode - Keyboard-dependant key code
	modifiers - The state of IBus.ModifierType keys like
		Shift, Control, etc.
Return:
	True - if successfully process the keyevent
	False - otherwise. The keyevent will be passed to X-Client

This function gets called whenever a key is pressed.
*/
func (e *IBusandodevel) ProcessKeyEvent(keyVal uint32, keyCode uint32, state uint32) (bool, *dbus.Error) {
	if e.isIgnoredKey(keyVal, state) {
		return false, nil
	}
	log.Printf("ProcessKeyEvent >  %c | keyCode 0x%04x keyVal 0x%04x | %d\n", rune(keyVal), keyCode, keyVal, len(keyPressChan))
	if e.isInputModeLTOpened {
		return e.ltProcessKeyEvent(keyVal, keyCode, state)
	}
	if e.inBackspaceWhiteList() {
		return e.bsProcessKeyEvent(keyVal, keyCode, state)
	}
	return e.preeditProcessKeyEvent(keyVal, keyCode, state)
}

func (e *IBusandodevel) FocusIn() *dbus.Error {
	log.Print("FocusIn.")
	var oldWmClasses = e.wmClasses
	e.wmClasses = x11GetFocusWindowClass()
	fmt.Printf("WM_CLASS=(%s)\n", e.wmClasses)

	e.RegisterProperties(e.propList)
	e.RequireSurroundingText()
	if oldWmClasses != e.wmClasses {
		e.resetBuffer()
		e.resetFakeBackspace()
	}
	return nil
}

func (e *IBusandodevel) FocusOut() *dbus.Error {
	log.Print("FocusOut.")
	//e.wmClasses = ""
	return nil
}

func (e *IBusandodevel) Reset() *dbus.Error {
	fmt.Print("Reset.\n")
	if e.checkInputMode(preeditIM) {
		e.commitPreedit(e.getPreeditString())
	}
	return nil
}

func (e *IBusandodevel) Enable() *dbus.Error {
	fmt.Print("Enable.")
	e.RequireSurroundingText()
	return nil
}

func (e *IBusandodevel) Disable() *dbus.Error {
	fmt.Print("Disable.")
	return nil
}

//@method(in_signature="vuu")
func (e *IBusandodevel) SetSurroundingText(text dbus.Variant, cursorPos uint32, anchorPos uint32) *dbus.Error {
	if !e.isSurroundingTextReady {
		//fmt.Println("Surrounding Text is not ready yet.")
		return nil
	}
	e.Lock()
	defer func() {
		e.Unlock()
		e.isSurroundingTextReady = false
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()
	if e.inBackspaceWhiteList() {
		var str = reflect.ValueOf(reflect.ValueOf(text.Value()).Index(2).Interface()).String()
		var s = []rune(str)
		if len(s) < int(cursorPos) {
			return nil
		}
		var cs = s[:cursorPos]
		fmt.Println("Surrounding Text: ", string(cs))
		e.preeditor.Reset()
		for i := len(cs) - 1; i >= 0; i-- {
			// workaround for spell checking
			if telex.IsPunctuationMark(cs[i]) && e.preeditor.CanProcessKey(cs[i]) {
				cs[i] = ' '
			}
			e.preeditor.ProcessKey(cs[i], telex.EnglishMode|telex.InReverseOrder)
		}
	}
	return nil
}

func (e *IBusandodevel) PageUp() *dbus.Error {
	if e.isInputModeLTOpened && e.inputModeLookupTable.PageUp() {
		e.updateInputModeLT()
	}
	return nil
}

func (e *IBusandodevel) PageDown() *dbus.Error {
	if e.isInputModeLTOpened && e.inputModeLookupTable.PageDown() {
		e.updateInputModeLT()
	}
	return nil
}

func (e *IBusandodevel) CursorUp() *dbus.Error {
	if e.isInputModeLTOpened && e.inputModeLookupTable.CursorUp() {
		e.updateInputModeLT()
	}
	return nil
}

func (e *IBusandodevel) CursorDown() *dbus.Error {
	if e.isInputModeLTOpened && e.inputModeLookupTable.CursorDown() {
		e.updateInputModeLT()
	}
	return nil
}

func (e *IBusandodevel) CandidateClicked(index uint32, button uint32, state uint32) *dbus.Error {
	if e.isInputModeLTOpened && e.inputModeLookupTable.SetCursorPos(index) {
		e.commitInputModeCandidate()
		e.closeInputModeCandidates()
	}
	return nil
}

func (e *IBusandodevel) SetCapabilities(cap uint32) *dbus.Error {
	e.capabilities = cap
	return nil
}

func (e *IBusandodevel) SetCursorLocation(x int32, y int32, w int32, h int32) *dbus.Error {
	return nil
}

func (e *IBusandodevel) SetContentType(purpose uint32, hints uint32) *dbus.Error {
	return nil
}

//@method(in_signature="su")
func (e *IBusandodevel) PropertyActivate(propName string, propState uint32) *dbus.Error {
	if propName == PropKeyAbout {
		exec.Command("xdg-open", HomePage).Start()
		return nil
	}
	if propName == PropKeyConfiguration {
		exec.Command("xdg-open", getConfigPath(e.engineName)).Start()
		return nil
	}

	if propName == PropKeyStdToneStyle {
		if propState == ibus.PROP_STATE_CHECKED {
			e.config.Flags |= telex.EstdToneStyle
		} else {
			e.config.Flags &= ^telex.EstdToneStyle
		}
	}
	if propName == PropKeyMouseCapturing {
		if propState == ibus.PROP_STATE_CHECKED {
			e.config.IBflags |= IBmouseCapturing
			startMouseCapturing()
		} else {
			e.config.IBflags &= ^IBmouseCapturing
			stopMouseCapturing()
		}
	}

	var charset, foundCs = getCharsetFromPropKey(propName)
	if foundCs && isValidCharset(charset) && propState == ibus.PROP_STATE_CHECKED {
		e.config.OutputCharset = charset
	}
	if _, found := e.config.InputMethodDefinitions[propName]; found && propState == ibus.PROP_STATE_CHECKED {
		e.config.InputMethod = propName
	}
	if propName != "-" {
		saveConfig(e.config, e.engineName)
	}
	e.propList = GetPropListByConfig(e.config)

	var inputMethod = telex.ParseInputMethod(e.config.InputMethodDefinitions, e.config.InputMethod)
	e.preeditor = telex.NewEngine(inputMethod, e.config.Flags)
	e.RegisterProperties(e.propList)
	return nil
}
