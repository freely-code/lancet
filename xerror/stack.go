// Copyright 2023 dudaodong@gmail.com. All rights reserved.
// Use of this source code is governed by MIT license

package xerror

import (
	"fmt"
	"io"
	"path"
	"runtime"
	"strconv"
	"strings"
)

// Stack contains function, file and line number info in the stack trace.
type Stack struct {
	Func string `json:"func"`
	File string `json:"file"`
	Line string `json:"line"`
}

// ---------------------------------------
// Stacktrace part is implemented based on copy of https://github.com/pkg/errors
//
// Copyright (c) 2015, Dave Cheney <dave@cheney.net>
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//
// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.
//
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
// FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
// DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
// SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
// CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
// OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

type frame uintptr
type stack []uintptr

// StackTrace is array of frame. It's exported for compatibility with github.com/pkg/errors
type StackTrace []frame

// pc returns the program counter for this frame;
// multiple frames may have the same PC value.
func (f frame) pc() uintptr { return uintptr(f) - 1 }

// file returns the full path to the file that contains the
// function for this Frame's pc.
func (f frame) file() string {
	fn := runtime.FuncForPC(f.pc())
	if fn == nil {
		return "unknown"
	}
	file, _ := fn.FileLine(f.pc())
	return file
}

// line returns the line number of source code of the
// function for this Frame's pc.
func (f frame) line() int {
	fn := runtime.FuncForPC(f.pc())
	if fn == nil {
		return 0
	}
	_, line := fn.FileLine(f.pc())
	return line
}

// name returns the name of this function, if known.
func (f frame) name() string {
	fn := runtime.FuncForPC(f.pc())
	if fn == nil {
		return "unknown"
	}
	return fn.Name()
}

// Format of frame formats the frame according to the fmt.Formatter interface.
//
//	%s    source file
//	%d    source line
//	%n    function name
//	%v    equivalent to %s:%d
//
// Format accepts flags that alter the printing of some verbs, as follows:
//
//	%+s   function name and path of source file relative to the compile time
//	      GOPATH separated by \n\t (<funcname>\n\t<path>)
//	%+v   equivalent to %+s:%d
func (f frame) Format(s fmt.State, verb rune) {
	switch verb {
	case 's':
		switch {
		case s.Flag('+'):
			_, _ = io.WriteString(s, f.name())
			_, _ = io.WriteString(s, "\n\t")
			_, _ = io.WriteString(s, f.file())
		default:
			_, _ = io.WriteString(s, path.Base(f.file()))
		}
	case 'd':
		_, _ = io.WriteString(s, strconv.Itoa(f.line()))
	case 'n':
		_, _ = io.WriteString(s, funcname(f.name()))
	case 'v':
		f.Format(s, 's')
		_, _ = io.WriteString(s, ":")
		f.Format(s, 'd')
	}
}

// MarshalText formats a stacktrace Frame as a text string. The output is the
// same as that of fmt.Sprintf("%+v", f), but without newlines or tabs.
func (f frame) MarshalText() ([]byte, error) {
	name := f.name()
	if name == "unknown" {
		return []byte(name), nil
	}
	return []byte(fmt.Sprintf("%s %s:%d", name, f.file(), f.line())), nil
}

// Format formats the stack of Frames according to the fmt.Formatter interface.
//
//	%s	lists source files for each Frame in the stack
//	%v	lists the source file and line number for each Frame in the stack
//
// Format accepts flags that alter the printing of some verbs, as follows:
//
//	%+v   Prints filename, function, and line number for each Frame in the stack.
func (st StackTrace) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		switch {
		case s.Flag('+'):
			for _, f := range st {
				_, _ = io.WriteString(s, "\n")
				f.Format(s, verb)
			}
		case s.Flag('#'):
			fmt.Fprintf(s, "%#v", []frame(st))
		default:
			st.formatSlice(s, verb)
		}
	case 's':
		st.formatSlice(s, verb)
	}
}

// formatSlice will format this StackTrace into the given buffer as a slice of
// Frame, only valid when called with '%s' or '%v'.
func (st StackTrace) formatSlice(s fmt.State, verb rune) {
	_, _ = io.WriteString(s, "[")
	for i, f := range st {
		if i > 0 {
			_, _ = io.WriteString(s, " ")
		}
		f.Format(s, verb)
	}
	_, _ = io.WriteString(s, "]")
}

func (s *stack) Format(st fmt.State, verb rune) {
	switch verb {
	case 'v':
		switch {
		case st.Flag('+'):
			for _, pc := range *s {
				f := frame(pc)
				fmt.Fprintf(st, "\n%+v", f)
			}
		}
	}
}

func (s *stack) StackTrace() StackTrace {
	frames := make([]frame, len(*s))
	for i := 0; i < len(frames); i++ {
		frames[i] = frame((*s)[i])
	}
	return frames
}

func callers() *stack {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(4, pcs[:])
	var st stack = pcs[0:n]
	return &st
}

// funcname removes the path prefix component of a function's name reported by func.Name().
func funcname(name string) string {
	i := strings.LastIndex(name, "/")
	name = name[i+1:]
	i = strings.Index(name, ".")
	return name[i+1:]
}
