package common

import (
	"bytes"
	"fmt"
)

var (
	PV         ProgramVersion
	Version    string
	CommitHash string
	BuildTime  string
)

func init() {
	PV.Version = Version
	PV.CommitHash = CommitHash
	PV.BuildTime = BuildTime
}

type ProgramVersion struct {
	Version    string `json:"version"`
	CommitHash string `json:"commit_hash"`
	BuildTime  string `json:"build_time"`
}

func (v ProgramVersion) Short() string {
	return fmt.Sprintf("v%s-%s-%s", v.Version, v.CommitHash, v.BuildTime)
}

func (v ProgramVersion) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("Author: Yihang Wang\n")
	buffer.WriteString("E-Mail: <wangyihanger@gmail.com>\n")
	buffer.WriteString(fmt.Sprintf("Version: v%s\n", v.Version))
	buffer.WriteString(fmt.Sprintf("Commit: %s\n", v.CommitHash))
	buffer.WriteString(fmt.Sprintf("Build Date: %s", v.BuildTime))
	return buffer.String()
}
