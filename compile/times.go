package compile

import "time"

// Times is where one compile's time went. Fill in by passing it in Context.Times.
type Times struct {
	Program, Copy, State, Prepare, Plugin, Check, Lower, Write, Declarations time.Duration
	FilesTransformed, FilesChecked, FilesLowered                             int
}
