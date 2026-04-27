package version

import "github.com/thomasgame/trojan-go/constant"

func Current() string {
	return constant.Version
}

func CommitID() string {
	return constant.Commit
}
