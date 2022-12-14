package graylog

import (
	"os"
	"path/filepath"
	"runtime/debug"
)

// SetFacility set Facility log filed used when generating a log message in
// GELF format. By default, it is filled in during initialization using the
// path to the  main module of the application.
//
// If you want to change this value, then this must be done before
// initializing the log.
func SetFacility(s string) {
	facility = s
}

var facility = initFacility()

func initFacility() string {
	var name string
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		name = buildInfo.Main.Path
	}

	if name == "" {
		name = filepath.Base(os.Args[0])
		name = name[:len(name)-len(filepath.Ext(name))]
	}

	if name == "." {
		name = ""
	}

	return name
}
