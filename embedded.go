package main

import "embed"

// SkillFiles is the embedded Oriyn skill tree. Lives at the module root so
// //go:embed can reach it — the directive cannot use `..` paths. cmd.Execute
// receives this FS and exposes it through the `oriyn skill install` command.
//
//go:embed all:skills/oriyn
var SkillFiles embed.FS
