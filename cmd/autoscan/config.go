package main

import (
	"github.com/kirsle/configdir"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
)

func defaultConfigPath() string {
	// get binary path
	bp := getBinaryPath()
	if dirIsWriteable(bp) == nil {
		return bp
	}

	// binary path is not write-able, use alternative path
	cp := configdir.LocalConfig("autoscan")
	if _, err := os.Stat(cp); os.IsNotExist(err) {
		if e := os.MkdirAll(cp, os.ModePerm); e != nil {
			panic("failed to create autoscan config directory")
		}
	}

	return cp
}

func getBinaryPath() string {
	// get current binary path
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		// get current working dir
		if dir, err = os.Getwd(); err != nil {
			panic("failed to determine current binary location")
		}
	}

	return dir
}

func dirIsWriteable(dir string) error {
	// credits: https://stackoverflow.com/questions/20026320/how-to-tell-if-folder-exists-and-is-writable
	return unix.Access(dir, unix.W_OK)
}
