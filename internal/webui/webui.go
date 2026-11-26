package webui

import "embed"

// DistFS 内嵌前端构建产物（由 frontend/ 通过 vite build 生成到 internal/webui/dist）。
// 任何对 internal/webui/dist 目录的修改都需重新执行 `npm run build`（见 frontend/ 与 Makefile）。
//
//go:embed all:dist
var DistFS embed.FS
