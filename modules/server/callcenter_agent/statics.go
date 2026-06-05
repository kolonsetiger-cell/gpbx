package callcenter_agent

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func setupStaticFiles(router *gin.Engine) {
	// 获取当前工作目录
	workDir, err := os.Getwd()
	if err != nil {
		return
	}

	// 静态文件目录（前端打包后的dist目录）
	staticDir := filepath.Join(workDir, "callcenter_agent_web")

	// 检查dist目录是否存在
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		return
	}

	// 静态文件服务
	router.Static("/assets", filepath.Join(staticDir, "assets"))
	router.Static("/images", filepath.Join(staticDir, "images"))

	// 其他静态资源
	router.StaticFile("/favicon.ico", filepath.Join(staticDir, "favicon.ico"))

	// 所有非API请求都返回index.html（支持前端路由）
	router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// API请求不处理
		if strings.HasPrefix(path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API not found"})
			return
		}

		// 静态文件请求不处理
		if strings.HasPrefix(path, "/assets") ||
			strings.HasPrefix(path, "/images") ||
			path == "/favicon.ico" {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
			return
		}

		// 返回index.html（前端路由处理）
		indexPath := filepath.Join(staticDir, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			c.File(indexPath)
		} else {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "index.html not found",
				"message": "请先构建前端项目: npm run build",
			})
		}
	})
}

func init() {
	statics = append(statics, setupStaticFiles)
}
