// Copyright 2015 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package lsof

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var (
	spacesReg  = regexp.MustCompile("\\s+")
	numbersReg = regexp.MustCompile("\\d+")
)

// Process defines a process using an open file. Properties here are strings
// for compatibility with different platforms.
type Process struct {
	PID        string
	Command    string
	UserID     string
	PortNumber int
}

// FileType defines the type of file in use by a process
type FileType string

const (
	FileTypeUnknown FileType = ""
	FileTypeDir     FileType = "DIR"
	FileTypeFile    FileType = "REG"
)

// FileDescriptor defines a file in use by a process
type FileDescriptor struct {
	FD   string
	Type FileType
	Name string
}

// ExecError is an error running lsof
type ExecError struct {
	command string
	args    []string
	output  string
	err     error
}

func (e ExecError) Error() string {
	return fmt.Sprintf("Error running %s %s: %s (%s)", e.command, e.args, e.err, e.output)
}

func fileTypeFromString(s string) FileType {
	switch s {
	case "DIR":
		return FileTypeDir
	case "REG":
		return FileTypeFile
	default:
		return FileTypeUnknown
	}
}

func (p *Process) fillField(s string) error {
	if s == "" {
		return errors.New("empty field")
	}
	values := spacesReg.Split(s, -1)
	portNumberByte := numbersReg.Find([]byte(values[8]))
	portNumber, err := strconv.Atoi(string(portNumberByte))
	if err == nil {
		p.PortNumber = portNumber
	}
	p.PID = values[1]
	p.Command = values[0]
	p.UserID = values[2]
	return nil
}

func (f *FileDescriptor) fillField(s string) error {
	// See Output for Other Programs at http://linux.die.net/man/8/lsof
	key := s[0]
	value := s[1:]
	switch key {
	case 't':
		f.Type = fileTypeFromString(value)
	case 'f':
		f.FD = value
	case 'n':
		f.Name = value
	default:
		// Skip unhandled field
	}

	return nil
}

func parseProcessLines(lines []string) (Process, error) {
	p := Process{}
	for _, line := range lines {
		err := p.fillField(line)
		if err != nil {
			return p, err
		}
	}
	return p, nil
}

func parseAppendProcessLines(processes []Process, linesChunk []string) ([]Process, []string, error) {
	if len(linesChunk) == 0 {
		return processes, linesChunk, nil
	}
	process, err := parseProcessLines(linesChunk)
	if err != nil {
		return processes, linesChunk, err
	}
	processesAfter := processes
	processesAfter = append(processesAfter, process)
	linesChunkAfter := []string{}
	return processesAfter, linesChunkAfter, nil
}

func parse(s string) ([]Process, error) {
	lines := strings.Split(s, "\n")
	linesChunk := []string{}
	processes := []Process{}
	if len(lines) > 1 {
		newLines := lines[1:]
		lines = newLines
	}
	var err error
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		// End of process, let's parse those lines
		if strings.HasPrefix(line, "p") && len(linesChunk) > 0 {
			processes, linesChunk, err = parseAppendProcessLines(processes, linesChunk)
			if err != nil {
				return nil, err
			}
		}
		linesChunk = append(linesChunk, line)
	}
	processes, _, err = parseAppendProcessLines(processes, linesChunk)
	if err != nil {
		return nil, err
	}
	return processes, nil
}

func Run() ([]Process, error) {
	// Some systems (Arch, Debian) install lsof in /usr/bin and others (centos)
	// install it in /usr/sbin, even though regular users can use it too. FreeBSD,
	// on the other hand, puts it in /usr/local/sbin. So do not specify absolute path.
	command := "lsof"
	args := append([]string{"-i", "-n", "-P", "-sTCP:LISTEN"})
	output, err := exec.Command(command, args...).Output()
	if err != nil {
		return nil, ExecError{command: command, args: args, output: string(output), err: err}
	}
	return parse(string(output))
}
