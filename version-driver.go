package driver

import _ "embed"

//go:embed VERSION
var versionString string

func DriverVersion() string {
	return versionString
}
