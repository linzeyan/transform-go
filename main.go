package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed web/*
var webFS embed.FS

func main() {
	r := gin.Default()

	// 取出 web/ 子目錄
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatal(err)
	}
	// 直接把 / 對應到嵌入式檔案系統
	// 會自動處理 Content-Type（含 .wasm）
	r.StaticFS("/", http.FS(sub))

	// SPA，需要把未知路由回傳 index.html
	r.NoRoute(func(c *gin.Context) {
		c.FileFromFS("index.html", http.FS(sub))
	})

	log.Println("listening on :8880")
	if err := r.Run(":8880"); err != nil {
		log.Fatal(err)
	}
}
