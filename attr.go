package anansi

import (
	"errors"
	"image"
	"os"
	"syscall"
)

var (
	errAttrForeignTerm = errors.New("invalid use of foreign terminal with anansi.Attr")
	errAttrNoFile      = errors.New("anansi.Attr.ioctl: no File set")
)

// Attr implements Context-ual manipulation and interrogation of terminal
// state, using the termios IOCTLs and ANSI control sequences where possible.
type Attr struct {
	*os.File
	active bool // true after Enter() before Exit()

	orig syscall.Termios
	cur  syscall.Termios
	raw  bool
	echo bool
}

// IsTerminal returns true only if the given file is attached to an interactive
// terminal.
func IsTerminal(f *os.File) bool {
	return Attr{f: f}.IsTerminal()
}

// IsTerminal returns true only if the underlying file is attached to an
// interactive terminal.
func (at Attr) IsTerminal() bool {
	_, err := at.getAttr()
	return err == nil
}

// Size reads and returns the current terminal size.
func (at Attr) Size() (size image.Point, err error) {
	return at.getSize()
}

// SetRaw controls whether the terminal should be in raw mode.
//
// Raw mode is suitable for full-screen terminal user interfaces, eliminating
// keyboard shortcuts for job control, echo, line buffering, and escape key
// debouncing.
func (at *Attr) SetRaw(raw bool) error {
	if raw == at.raw {
		return nil
	}
	at.raw = raw
	if at.active {
		at.cur = at.modifyTermios(at.orig)
		return at.setAttr(at.cur)
	}
	return nil
}

// SetEcho toggles input echoing mode, which is off by default in raw mode, and
// on in normal mode.
func (at *Attr) SetEcho(echo bool) error {
	if echo == at.echo {
		return nil
	}
	at.echo = echo
	if at.active {
		if echo {
			at.cur.Lflag |= syscall.ECHO
		} else {
			at.cur.Lflag &^= syscall.ECHO
		}
		return at.setAttr(at.cur)
	}
	return nil
}

func (at Attr) modifyTermios(attr syscall.Termios) syscall.Termios {
	if at.raw {
		// TODO read things like antirez's kilo notes again

		// TODO naturalize / decompose
		attr.Iflag &^= syscall.BRKINT | syscall.ICRNL | syscall.INPCK | syscall.ISTRIP | syscall.IXON
		attr.Oflag &^= syscall.OPOST
		attr.Cflag &^= syscall.CSIZE | syscall.PARENB
		attr.Cflag |= syscall.CS8
		attr.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.IEXTEN | syscall.ISIG
		attr.Cc[syscall.VMIN] = 1
		attr.Cc[syscall.VTIME] = 0

	}
	if at.echo {
		attr.Lflag |= syscall.ECHO
	} else {
		attr.Lflag &^= syscall.ECHO
	}
	return attr
}

// Enter records original termios attributes, and then applies termios
// attributes.
func (at *Attr) Enter(term *Term) (err error) {
	if at != &term.Attr {
		return errAttrForeignTerm
	}
	if at.orig, err = at.getAttr(); err == nil {
		at.cur = at.modifyTermios(at.orig)
		err = at.setAttr(at.cur)
		if err == nil {
			at.active = true
		}
	}
	return err
}

// Exit restores termios attributes only.
func (at *Attr) Exit(term *Term) error {
	if at == &term.Attr {
		at.active = false
		return at.setAttr(at.orig)
	}
	return nil
}

func (at Attr) ioctl(request, arg1, arg2, arg3, arg4 uintptr) error {
	if at.File == nil {
		return errAttrNoFile
	}
	if _, _, e := syscall.Syscall6(syscall.SYS_IOCTL, at.File.Fd(), request, arg1, arg2, arg3, arg4); e != 0 {
		return e
	}
	return nil
}
