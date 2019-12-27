package utils

import "strings"

const maxNameLen = 63

// StripName strips the extra characters from the name
// Since Custom Resources only support names upto 63 chars
// so this trims the rest of the trailing chars
func StripName(name string) string {
	name = strings.ToLower(name)
	if len(name) > maxNameLen {
		name = name[:maxNameLen]
	}
	return name
}
