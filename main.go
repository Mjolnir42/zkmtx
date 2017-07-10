/*-
 * Copyright © 2017, Jörg Pernfuß <code.jpe@gmail.com>
 * All rights reserved.
 *
 * Use of this source code is governed by a 2-clause BSD license
 * that can be found in the LICENSE file.
 */

package main // import "github.com/mjolnir42/zkrun"

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/client9/reopen"
	"github.com/droundy/goopt"
	shellwords "github.com/mattn/go-shellwords"
	"github.com/mjolnir42/erebos"
	"github.com/samuel/go-zookeeper/zk"
)

var lockPath, zkrunPath, zkrunVersion string
var conf *Config
var jobSpec *JobSpec
var logInitialized bool

func init() {
	// Discard logspam from Zookeeper library
	erebos.DisableZKLogger()

	// set standard logger options
	erebos.SetLogrusOptions()

	// set goopt information
	goopt.Version = zkrunVersion
	goopt.Suite = `zkRUN`
	goopt.Summary = `Command execution under distributed mutex lock`
	goopt.Author = `Jörg Pernfuß`
	goopt.Description = func() string {
		return "zkRUN"
	}
}

func main() {
	os.Exit(run())
}

func run() int {
	// parse command line flags
	cliConfPath := goopt.String([]string{`-c`, `--config`},
		`/etc/zkrun/zkrun.conf`, `Configuration file`)
	jobConfPath := goopt.String([]string{`-j`, `--job`},
		``, `Job name to run command`)
	goopt.Parse(nil)

	// validate cli input
	validJob(jobConfPath)

	// read runtime configuration
	conf = &Config{}
	if err := conf.FromFile(*cliConfPath); err != nil {
		assertOK(fmt.Errorf("Could not open configuration: %s", err))
	}

	// read job specification
	if !filepath.IsAbs(*jobConfPath) {
		if !strings.HasSuffix(*jobConfPath, `.conf`) {
			*jobConfPath = *jobConfPath + `.conf`
		}
		*jobConfPath = filepath.Join(
			`/etc/zkrun/jobspec`,
			*jobConfPath,
		)
	}
	jobSpec = &JobSpec{}
	if err := jobSpec.FromFile(*jobConfPath); err != nil {
		assertOK(fmt.Errorf("Could not open configuration: %s", err))
	}

	// validate we can fork to the requested user
	validUser()
	validSyncGroup()
	validExitPolicy(jobSpec.ExitPolicy)

	// setup logfile
	if lfh, err := reopen.NewFileWriter(conf.LogFile); err != nil {
		assertOK(fmt.Errorf("Unable to open logfile: %s", err))
	} else {
		logrus.SetOutput(lfh)
		logInitialized = true
	}
	logrus.Infoln(`Starting zkRUN`)

	conn, chroot := connect(conf.Ensemble)
	defer conn.Close()
	logrus.Infoln(`Configured zookeeper chroot:`, chroot)

	// ensure fixed node hierarchy exists
	if !zkHier(conn, filepath.Join(chroot, `zkrun`), true) {
		return 1
	}

	// ensure required nodes exist
	zkrunPath = filepath.Join(chroot, `zkrun`, conf.SyncGroup)
	if !zkCreatePath(conn, zkrunPath, true) {
		return 1
	}

	zkrunPath = filepath.Join(zkrunPath, filepath.Base(
		strings.TrimSuffix(*jobConfPath, `.conf`)))
	if !zkCreatePath(conn, zkrunPath, true) {
		return 1
	}

	lockPath = filepath.Join(zkrunPath, `lock`)
	if !zkCreatePath(conn, lockPath, true) {
		return 1
	}

	for {
		leaderChan, errChan := zkLeaderLock(conn)

		block := make(chan error)
		select {
		case <-errChan:
			return 1
		case <-leaderChan:
			go leader(conn, block)
		}
		err := <-block
		if jobSpec.ExitPolicy == `reaquire-lock` {
			if err != nil {
				logrus.Errorln(err.Error())
			}
			continue
		}
		// ExitPolicy: terminate
		if errorOK(err) {
			return 1
		}
		logrus.Infof("Shutting down")
		return 0
	}
}

func leader(conn *zk.Conn, block chan error) {
	logrus.Infoln("Leader election has been won")

	var err error

	active := filepath.Join(zkrunPath, `active`)
	if !zkCreateEph(conn, active) {
		close(block)
		return
	}

	cmdSlice, err := shellwords.Parse(jobSpec.Command)
	if sendError(err, block) {
		return
	}
	if len(cmdSlice) == 0 {
		close(block)
		return
	}
	cmd := exec.Command(cmdSlice[0], cmdSlice[1:]...)
	logrus.Infoln("Running command")

	if conf.User != `` {
		user, uerr := user.Lookup(conf.User)
		if sendError(uerr, block) {
			return
		}
		uid, uerr := strconv.Atoi(user.Uid)
		if sendError(uerr, block) {
			return
		}
		gid, uerr := strconv.Atoi(user.Gid)
		if sendError(uerr, block) {
			return
		}
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Credential: &syscall.Credential{
				Uid:    uint32(uid),
				Gid:    uint32(gid),
				Groups: []uint32{},
			},
		}
	}

	wait := make(chan error)
	cmd.Stdout = ioutil.Discard
	cmd.Stderr = ioutil.Discard
	err = cmd.Start()
	// if we die, so does the process...
	defer cmd.Process.Kill()

	if sendError(err, block) {
		return
	}
	go func(w chan error) {
		err := cmd.Wait()
		w <- err
		close(w)
	}(wait)

	// check if the command runs long enough to fulfill the
	// start success delay
	select {
	case err := <-wait:
		if sendError(err, block) {
			return
		}
	case <-time.After(jobSpec.StartSuccess):
	}

	// run all AfterStart commands
	for i := range jobSpec.AfterStart {
		asLine, asErr := shellwords.Parse(jobSpec.AfterStart[i])
		if sendError(asErr, block) {
			cmd.Process.Kill()
			return
		}
		asCmd := exec.Command(asLine[0], asLine[1:]...)
		asCmd.Stdout = ioutil.Discard
		asCmd.Stderr = ioutil.Discard
		asErr = asCmd.Run()
		if sendError(asErr, block) {
			cmd.Process.Kill()
			return
		}
	}

	fault := false
	// wait for process to exit
	err = <-wait
	switch {
	case err == nil:
	case err.(*exec.ExitError).Success():
	default:
		fault = true
	}

	if fault && conf.ExitPolicy == `run-command` {
		// run all AfterExitFail commands
		for i := range jobSpec.AfterExitFail {
			aefLine, aefErr := shellwords.Parse(
				jobSpec.AfterExitFail[i])
			errorOK(aefErr)
			aefCmd := exec.Command(aefLine[0], aefLine[1:]...)
			aefCmd.Stdout = ioutil.Discard
			aefCmd.Stderr = ioutil.Discard
			aefErr = aefCmd.Run()
			errorOK(aefErr)
		}
	}

	// run all AfterExit commands
	for i := range jobSpec.AfterExit {
		aeLine, aeErr := shellwords.Parse(
			jobSpec.AfterExit[i])
		errorOK(aeErr)
		aeCmd := exec.Command(aeLine[0], aeLine[1:]...)
		aeCmd.Stdout = ioutil.Discard
		aeCmd.Stderr = ioutil.Discard
		aeErr = aeCmd.Run()
		errorOK(aeErr)
	}
	if fault {
		block <- fmt.Errorf(`Command exited with failure code`)
	}
	close(block)
}

// vim: ts=4 sw=4 sts=4 noet fenc=utf-8 ffs=unix
