// Package version provides version output for ghard
package version

import "fmt"

// Version is set at build time using -ldflags
var Version = "dev"

// ShowVersion prints the current version information
func ShowVersion() {
	fmt.Println(Version)
}
