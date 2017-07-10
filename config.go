/*-
 * Copyright © 2017, Jörg Pernfuß <code.jpe@gmail.com>
 * All rights reserved.
 *
 * Use of this source code is governed by a 2-clause BSD license
 * that can be found in the LICENSE file.
 */

package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"time"

	ucl "github.com/nahanni/go-ucl"
)

// Config holds the runtime configuration which is expected to be
// read from a UCL formatted file
type Config struct {
	// Zookeeper connect string host:port,host:port/chroot
	Ensemble string `json:"ensemble"`
	// Name of the syncgroup
	SyncGroup string `json:"sync.group"`
	// Full path of the logfile
	LogFile string `json:"log.file"`
	// User to run Command under. Only root can switch users. Empty
	// default to the same user as zkrun
	User string `json:"run.as.user"`
}

// JobSpec holds the runtime configuration for a specific job which
// is expected to be read from a UCL formatted file
type JobSpec struct {
	// Command is the commandline to run
	Command string `json:"command"`
	// Parsed as time.Duration, the amount of time the command
	// must be running after startup before the start is considered
	// a success
	StartSuccess time.Duration `json:"start.success.delay,string"`
	// ExitPolicy determines what to do if the process exits. Options
	// are:
	//	- reaquire-lock, to try and run the command again
	//	- run-command, to execute after.exit.failure commands
	//	- terminate, to terminate zkrun
	ExitPolicy string `json:"exit.policy"`
	// Commandlines to run after a successful start. These must succeed
	// or zkrun exits.
	AfterStart []string `json:"after.start.success"`
	// Commandlines to run if command exits with exitcode != 0 and
	// ExitPolicy is set to run-command
	AfterExitFail []string `json:"after.exit.failure"`
	// Commandlines to always run if Command exits, regardless of
	// exitcode or ExitPolicy.
	AfterExit []string `json:"after.exit.always"`
}

// FromFile sets Config c based on the file contents
func (c *Config) FromFile(fname string) error {
	var (
		file, uclJSON []byte
		err           error
		fileBytes     *bytes.Buffer
		parser        *ucl.Parser
		uclData       map[string]interface{}
	)
	if fname, err = filepath.Abs(fname); err != nil {
		return err
	}
	if fname, err = filepath.EvalSymlinks(fname); err != nil {
		return err
	}
	if file, err = ioutil.ReadFile(fname); err != nil {
		return err
	}

	fileBytes = bytes.NewBuffer(file)
	parser = ucl.NewParser(fileBytes)
	if uclData, err = parser.Ucl(); err != nil {
		return err
	}

	if uclJSON, err = json.Marshal(uclData); err != nil {
		return err
	}
	return json.Unmarshal(uclJSON, &c)
}

// FromFile sets JobSpec j based on the file contents
func (j *JobSpec) FromFile(fname string) error {
	var (
		file, uclJSON []byte
		err           error
		fileBytes     *bytes.Buffer
		parser        *ucl.Parser
		uclData       map[string]interface{}
	)
	if fname, err = filepath.Abs(fname); err != nil {
		return err
	}
	if fname, err = filepath.EvalSymlinks(fname); err != nil {
		return err
	}
	if file, err = ioutil.ReadFile(fname); err != nil {
		return err
	}

	fileBytes = bytes.NewBuffer(file)
	parser = ucl.NewParser(fileBytes)
	if uclData, err = parser.Ucl(); err != nil {
		return err
	}

	if uclJSON, err = json.Marshal(uclData); err != nil {
		return err
	}
	return json.Unmarshal(uclJSON, &j)
}

// vim: ts=4 sw=4 sts=4 noet fenc=utf-8 ffs=unix
