package autoscan

import (
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
)

func GetDefaultConfigPath() string {
	// get binary path
	bp := getCurrentBinaryPath()
	if dirIsWriteable(bp) == nil {
		return bp
	}

	// binary path is not write-able, use alternative path
	uhp, err := os.UserHomeDir()
	if err != nil {
		panic("failed to determine current user home directory")
	}

	// set autoscan path inside user home dir
	chp := filepath.Join(uhp, ".config", "autoscan")
	if _, err := os.Stat(chp); os.IsNotExist(err) {
		if e := os.MkdirAll(chp, os.ModePerm); e != nil {
			panic("failed to create autoscan config directory")
		}
	}

	return chp
}

/* Private */

func getCurrentBinaryPath() string {
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
