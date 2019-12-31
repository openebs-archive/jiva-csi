package utils

import "strings"

const maxNameLen = 43

// StripName strips the extra characters from the name
// Since Custom Resources only support names upto 63 chars
// so this trims the rest of the trailing chars and it generates
// the controller-revision hash of by appending more 10 chars
// after appending `-jiva-rep-` so total 20 chars must be stripped
func StripName(name string) string {
	name = strings.ToLower(name)
	if len(name) > maxNameLen {
		name = name[:maxNameLen]
	}

	if strings.HasSuffix(name, "-") {
		name = name[:len(name)-1]
	}
	return name
}
