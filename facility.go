package graylog

import (
	"os"
	"path/filepath"
	"runtime/debug"
)

// Used when generating a log message in GELF format.
// By default, it is filled in during initialization using the path to the
// main module of the application.
//
// If you want to change this value, then this must be done before
// initializing the log.
var Facility string

func init() {
	var name string
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		name = buildInfo.Main.Path
	} else {
		name = filepath.Base(os.Args[0])
		name = name[:len(name)-len(filepath.Ext(name))]
	}

	if name == "." {
		name = ""
	}

	Facility = name
}
