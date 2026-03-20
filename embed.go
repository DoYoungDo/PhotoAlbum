package main

import "embed"

// webFS 包含所有前端资源，通过 embed 编译进二进制
//
//go:embed web/static web/templates
var webFS embed.FS
