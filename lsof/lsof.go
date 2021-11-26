// Copyright 2015 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package lsof

import (
	"errors"
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

func parseAppendProcessLines(processes []Process, linesChunk []string) ([]Process, error) {
	if len(linesChunk) == 0 {
		return processes, nil
	}

	for _, line := range linesChunk {
		p := Process{}
		err := p.fillField(line)
		if err != nil {
			continue
		}
		processes = append(processes, p)
	}
	return processes, nil
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

		if !strings.Contains(line, "LISTEN") {
			continue
		}

		linesChunk = append(linesChunk, line)
	}
	processes, err = parseAppendProcessLines(processes, linesChunk)
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
	args := append([]string{"-i", "-n", "-P"})
	output, err := exec.Command(command, args...).Output()
	if err != nil {
		return nil, err
	}
	return parse(string(output))
}
