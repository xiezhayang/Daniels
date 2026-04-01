package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/xiezhayang/Daniels/internal/config"
	"github.com/xiezhayang/Daniels/internal/proxy"
	"github.com/xiezhayang/Daniels/internal/window"
)

func main() {
	cfg := config.Load()

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// 健康检查
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 实验窗口记录（给 Grafana 注释/NAB 红窗）
	store := window.NewStore()
	winHandler := window.NewHandler(store)

	r.GET("/api/v1/windows", winHandler.ListWindows)
	r.DELETE("/api/v1/windows", winHandler.ClearWindows)

	// Carve / Omar 反向代理
	carveProxy := proxy.NewReverseProxy(cfg.CarveBaseURL, proxy.Options{
		WrapResponse: true,
		Tag:          "carve",
	})
	omarProxy := proxy.NewReverseProxy(cfg.OmarBaseURL, proxy.Options{
		WrapResponse: true,
		Tag:          "omar",
		OnBeforeProxy: func(ctx *proxy.ProxyContext) {
			// 记录实验窗口：拦截 Omar 创建实验请求
			// POST /api/v1/experiments
			if ctx.Method == "POST" && ctx.Path == "/api/v1/experiments" {
				// 非阻塞解析，失败不影响转发
				window.TryRecordFromCreateExperiment(ctx.RawBody, store)
			}
		},
	})
	grafanaProxy := proxy.NewReverseProxy(cfg.GrafanaBaseURL, proxy.Options{
		WrapResponse:   false,
		Tag:            "grafana",
		LocationPrefix: "/obs",
	})
	prometheusProxy := proxy.NewReverseProxy(cfg.PrometheusBaseURL, proxy.Options{
		WrapResponse:   false,
		Tag:            "prometheus",
		LocationPrefix: "/obs/prometheus",
	})

	// 观测系统反代（给前端同域跳转）
	r.Any("/obs/grafana/*path", grafanaProxy)
	r.Any("/obs/prometheus/*path", prometheusProxy)
	// 前端统一走 /prod-api/carve/*, /prod-api/omar/*
	r.Any("/prod-api/carve/*path", carveProxy)
	r.Any("/prod-api/omar/*path", omarProxy)

	addr := ":" + cfg.Port
	log.Printf("[gateway] listen on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
