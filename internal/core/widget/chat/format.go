package chat

import "regexp"

var (
	dayRe     = regexp.MustCompile(`^##\s+(\d{4}-\d{2}-\d{2})\s*$`)
	messageRe = regexp.MustCompile(`^@([\p{L}\p{N}._/-]+)\s+(\d{1,2}:\d{2})\s+(.*)$`)
)
