package common

import (
	"bytes"
	"fmt"
)

var (
	// PV is the current version object of the program
	PV ProgramVersion
	// Version is the current version of the program
	Version string
	// CommitHash is the current commit hash of the program
	CommitHash string
	// BuildTime is the current build time of the program
	BuildTime string
)

func init() {
	PV.Version = Version
	PV.CommitHash = CommitHash
	PV.BuildTime = BuildTime
}

// ProgramVersion is the version object of the program
type ProgramVersion struct {
	Version    string `json:"version"`
	CommitHash string `json:"commit_hash"`
	BuildTime  string `json:"build_time"`
}

// Short returns the short version of the program
func (v ProgramVersion) Short() string {
	return fmt.Sprintf("v%s-%s-%s", v.Version, v.CommitHash, v.BuildTime)
}

// String returns the verbose version of the program
func (v ProgramVersion) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("Author: Yihang Wang\n")
	buffer.WriteString("E-Mail: <wangyihanger@gmail.com>\n")
	buffer.WriteString(fmt.Sprintf("Version: v%s\n", v.Version))
	buffer.WriteString(fmt.Sprintf("Commit: %s\n", v.CommitHash))
	buffer.WriteString(fmt.Sprintf("Build Date: %s", v.BuildTime))
	return buffer.String()
}
