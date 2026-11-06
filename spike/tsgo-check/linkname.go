package main

import (
	"context"
	"fmt"
	_ "unsafe" // required by go:linkname

	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/internal/compiler"
)

// linknameGetStringLiteralType reaches an unexported method the way tsgolint's shims do.
// The spike uses the patch for real work. This only proves the linker accepts the mechanism.
//
//go:linkname linknameGetStringLiteralType github.com/microsoft/TypeScript/tsc/internal/checker.(*Checker).getStringLiteralType
func linknameGetStringLiteralType(c *checker.Checker, value string) *checker.Type

func checkLinkname(program *compiler.Program) {
	c, done := program.GetTypeChecker(context.Background())
	defer done()
	viaLinkname := linknameGetStringLiteralType(c, "spike")
	viaPatch := c.GetStringLiteralType("spike")
	fmt.Printf("\ngo:linkname: getStringLiteralType returned %s, same type as the patched export: %v\n", c.TypeToString(viaLinkname), viaLinkname == viaPatch)
}
