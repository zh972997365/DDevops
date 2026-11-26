package router

import (
	"io/fs"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"ddevops/internal/auth"
	"ddevops/internal/handler"
	"ddevops/internal/middleware"
	"ddevops/internal/webui"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// SetupRouter 构建并返回 Gin 引擎：
//   - /api 分组承载全部后端接口（含登录校验与管理员校验中间件）
//   - 其余路径由内嵌的前端 SPA（internal/webui/dist）提供，未知路由回退到 index.html
func SetupRouter(apiHandler *handler.APIHandler, authSvc *auth.AuthService) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// 前端同源加载，token 通过 Authorization header 传递（非 cookie），
	// 因此无需 AllowCredentials；原先的 "*" + AllowCredentials:true 组合在浏览器中无效。
	r.Use(cors.New(cors.Config{
		AllowOrigins:  []string{"*"},
		AllowMethods:  []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:  []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders: []string{"Content-Length"},
		MaxAge:        12 * time.Hour,
	}))

	registerAPI(r, apiHandler, authSvc)
	registerSPA(r)

	return r
}

func registerAPI(r *gin.Engine, apiHandler *handler.APIHandler, authSvc *auth.AuthService) {
	api := r.Group("/api")
	{
		api.POST("/login", apiHandler.Login)

		auth := api.Group("")
		auth.Use(middleware.AuthMiddleware(authSvc))
		{
			auth.POST("/logout", apiHandler.Logout)

			auth.GET("/jobs", apiHandler.ListJobs)
			auth.GET("/jobs/:id/git-tags", apiHandler.GetJobGitTags)
			auth.POST("/jobs/:id/build", apiHandler.BuildJob)

			auth.GET("/environments", apiHandler.ListEnvironments)
			// 服务器精简列表（仅 id/name）：普通用户的任务列表要显示所属服务器名称
			auth.GET("/servers/options", apiHandler.ListServerOptions)

			auth.GET("/tasks", apiHandler.ListTasks)
			auth.POST("/tasks", apiHandler.CreateTask)
			auth.POST("/tasks/:id/cancel", apiHandler.CancelTask)

			auth.GET("/history", apiHandler.ListHistory)
			auth.GET("/history/stats", apiHandler.HistoryStats)
			auth.GET("/overview", apiHandler.GetOverview)

			admin := auth.Group("")
			admin.Use(middleware.AdminMiddleware())
			{
				admin.POST("/jobs", apiHandler.CreateJob)
				admin.PUT("/jobs/:id", apiHandler.UpdateJob)
				admin.DELETE("/jobs/:id", apiHandler.DeleteJob)

				admin.DELETE("/tasks/:id", apiHandler.DeleteTask)

				admin.GET("/permissions", apiHandler.ListPermissions)
				admin.POST("/permissions", apiHandler.CreatePermission)
				admin.PUT("/permissions/:userId", apiHandler.UpdatePermission)
				admin.DELETE("/permissions/:userId", apiHandler.DeletePermission)

				admin.GET("/servers", apiHandler.ListServers)
				admin.POST("/servers", apiHandler.CreateServer)
				admin.PUT("/servers/reorder", apiHandler.ReorderServers)
				admin.PUT("/servers/:id", apiHandler.UpdateServer)
				admin.DELETE("/servers/:id", apiHandler.DeleteServer)

				admin.POST("/environments", apiHandler.CreateEnvironment)
				admin.PUT("/environments/reorder", apiHandler.ReorderEnvironments)
				admin.PUT("/environments/:id", apiHandler.UpdateEnvironment)
				admin.DELETE("/environments/:id", apiHandler.DeleteEnvironment)

				// 通讯录接口收紧为管理员专用：普通用户无需也不应拉取/同步钉钉通讯录。
				admin.GET("/dingtalk/users", apiHandler.SearchDingTalkUsers)
				admin.POST("/dingtalk/sync", apiHandler.SyncDingTalkUsers)
			}
		}
	}
}

// registerSPA 将内嵌的前端构建产物挂载为静态资源，并对非 /api 的未知路由回退到 index.html，
// 以支持 Vue 单页应用的客户端路由（hash 模式）。
func registerSPA(r *gin.Engine) {
	sub, err := fs.Sub(webui.DistFS, "dist")
	if err != nil {
		log.Printf("加载前端静态资源失败: %v", err)
		return
	}
	fileServer := http.FileServer(http.FS(sub))

	// 预先读取 SPA 入口，避免回退时再次 IO
	indexHTML, err := webui.DistFS.ReadFile("dist/index.html")
	if err != nil {
		log.Printf("读取前端入口失败: %v", err)
		return
	}

	r.NoRoute(func(c *gin.Context) {
		reqPath := c.Request.URL.Path

		// API 路径统一返回 JSON 404，避免被 SPA 回退吞掉
		if strings.HasPrefix(reqPath, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		cleanPath := strings.TrimPrefix(path.Clean(reqPath), "/")

		// 若文件真实存在于 dist，则直接提供静态资源（含正确的 Content-Type）
		if cleanPath != "" {
			if f, openErr := sub.Open(cleanPath); openErr == nil {
				if info, statErr := f.Stat(); statErr == nil && !info.IsDir() {
					f.Close()
					// 文件名带内容 hash，可以放心长缓存
					if strings.HasPrefix(cleanPath, "assets/") {
						c.Header("Cache-Control", "public, max-age=31536000, immutable")
					}
					fileServer.ServeHTTP(c.Writer, c.Request)
					return
				}
				f.Close()
			}
		}

		// 否则回退到 index.html（SPA 入口），直接写入避免 http.FileServer 的目录 301 重定向。
		// 入口必须禁缓存：否则前端重新构建后浏览器仍用旧 index.html，引用的旧 JS 让改动"看起来没生效"。
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})
}

// StartHealthCheck 在独立端口提供 /health 健康检查。
func StartHealthCheck(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	return http.ListenAndServe(addr, mux)
}
