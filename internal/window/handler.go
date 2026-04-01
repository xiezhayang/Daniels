package window

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	store *Store
}

func NewHandler(s *Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) ListWindows(c *gin.Context) {
	c.JSON(200, gin.H{
		"code": "00000",
		"data": h.store.List(),
		"msg":  "ok",
	})
}

func (h *Handler) ClearWindows(c *gin.Context) {
	h.store.Clear()
	c.JSON(200, gin.H{
		"code": "00000",
		"data": true,
		"msg":  "ok",
	})
}

// Omar 创建实验请求体（按你现有 handler.go）
type createExperimentReq struct {
	TargetType      string `json:"target_type"`
	TargetName      string `json:"target_name"`
	TargetNamespace string `json:"target_namespace"`
	ExperimentType  string `json:"experiment_type"`
	Duration        int    `json:"duration"` // 秒
}

func TryRecordFromCreateExperiment(raw []byte, store *Store) {
	var req createExperimentReq
	if err := json.Unmarshal(raw, &req); err != nil {
		return
	}
	if req.ExperimentType == "" || req.TargetType == "" {
		return
	}

	dur := req.Duration
	if dur <= 0 {
		// Omar 允许不传 duration（会随机），网关无法知道真实结束时间
		// 这里先给一个默认窗口，后面你可以升级成“轮询 active 实验纠偏”
		dur = 120
	}

	now := time.Now().UTC()
	w := Window{
		ID:             fmt.Sprintf("%s-%d", req.ExperimentType, now.UnixNano()),
		Source:         "omar",
		ExperimentType: req.ExperimentType,
		TargetType:     req.TargetType,
		TargetName:     req.TargetName,
		TargetNS:       req.TargetNamespace,
		StartAt:        now,
		EndAt:          now.Add(time.Duration(dur) * time.Second),
		DurationSec:    dur,
	}
	store.Add(w)
}
