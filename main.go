// trae-signin-web 后端入口：加载存储 → 启动定时调度 → 暴露 API + 静态前端。
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"trae-signin-web/internal/api"
	"trae-signin-web/internal/checkin"
	"trae-signin-web/internal/scheduler"
	"trae-signin-web/internal/store"
	"trae-signin-web/internal/upstream"
	"trae-signin-web/internal/webauth"
)

func main() {
	// 默认端口 9090，可被 PORT 环境变量覆盖（Docker 习惯）
	defaultAddr := ":9090"
	if p := os.Getenv("PORT"); p != "" {
		defaultAddr = ":" + p
	}
	var (
		addr = flag.String("addr", defaultAddr, "监听地址")
		data = flag.String("data", "./data/data.json", "数据文件路径")
		dist = flag.String("dist", "./web/dist", "前端静态文件目录")
	)
	flag.Parse()

	st, err := store.New(*data)
	if err != nil {
		log.Fatalf("加载存储失败: %v", err)
	}
	// Web 登录认证（auth.json 与 data.json 同目录）
	authPath := filepath.Join(filepath.Dir(*data), "auth.json")
	authMgr, err := webauth.New(authPath)
	if err != nil {
		log.Fatalf("加载认证数据失败: %v", err)
	}
	up := upstream.New()
	svc := checkin.New(st, up)
	sched := scheduler.New(st, svc)
	sched.Start()
	defer sched.Stop()

	mux := http.NewServeMux()
	mux.Handle("/api/", api.NewHandler(api.Deps{Store: st, Up: up, Svc: svc, Sched: sched, Auth: authMgr}))

	// 静态前端（SPA）：dist 存在则托管，否则仅 API
	if info, err := os.Stat(*dist); err == nil && info.IsDir() {
		fs := http.FileServer(http.Dir(*dist))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// API 请求不走静态
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.NotFound(w, r)
				return
			}
			// SPA 回退：文件不存在则返回 index.html
			p := filepath.Join(*dist, filepath.Clean(r.URL.Path))
			if _, err := os.Stat(p); os.IsNotExist(err) {
				r2 := r.Clone(r.Context())
				r2.URL.Path = "/"
				fs.ServeHTTP(w, r2)
				return
			}
			fs.ServeHTTP(w, r)
		})
		log.Printf("前端静态文件目录: %s", *dist)
	} else {
		log.Printf("前端目录 %s 不存在，仅启动 API（访问 /api/*）", *dist)
	}

	log.Printf("trae-signin-web 启动于 %s", *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatalf("服务退出: %v", err)
	}
}
