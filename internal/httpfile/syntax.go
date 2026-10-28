// Package httpfile reads and sends the requests of a .http file, the plain
// text format that resterm, the VS Code REST Client and JetBrains share.
// Directives in comments ("# @assert status == 200") check the response,
// carry values to the next request and run hooks around it.
//
// The files: internal/httpfile/syntax reads a file, vars.go fills in
// {{variables}} from the file, the environment and earlier responses,
// run.go sends a request and checks what came back, and log.go writes it
// down for the board and the agent.
package httpfile

import "github.com/LeeSwallow/stickypane/internal/httpfile/syntax"

// The file's grammar is internal/httpfile/syntax's, which reads without
// sending; these are its names here, for the code that sends.
type (
	Var       = syntax.Var
	Capture   = syntax.Capture
	Request   = syntax.Request
	WebSocket = syntax.WebSocket
	Step      = syntax.Step
	GRPC      = syntax.GRPC
	File      = syntax.File
)

// Parse reads a .http file.
func Parse(text string) File { return syntax.Parse(text) }
