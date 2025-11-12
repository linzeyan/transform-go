package main

import (
	"embed"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed web/*
var webFS embed.FS

func main() {
	r := gin.Default()
	// 靜態頁面
	r.GET("/", func(c *gin.Context) {
		f, err := webFS.ReadFile("web/index.html")
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", f)
	})
	r.GET("/app.js", serveAsset("web/app.js", "text/javascript", true))
	r.GET("/style.css", serveAsset("web/style.css", "text/css", true))
	r.GET("/wasm_exec.js", serveAsset("web/wasm_exec.js", "text/javascript", true))
	r.GET("/app.wasm", serveAsset("web/app.wasm", "application/wasm", false))
	r.GET("/favicon.svg", serveAsset("web/favicon.svg", "image/svg+xml", false))

	log.Println("listening on :8880")
	if err := r.Run(":8880"); err != nil {
		log.Fatal(err)
	}
}

func serveAsset(path, mime string, withCharset bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		b, err := webFS.ReadFile(path)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		ct := mime
		if withCharset {
			ct += "; charset=utf-8"
		}
		c.Data(http.StatusOK, ct, b)
	}
}
