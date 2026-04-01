package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/xiezhayang/Daniels/internal/response"
)

type Options struct {
	WrapResponse   bool
	Tag            string
	OnBeforeProxy  func(*ProxyContext)
	LocationPrefix string
}

type ProxyContext struct {
	Method  string
	Path    string
	RawBody []byte
}

func NewReverseProxy(targetBase string, opts Options) gin.HandlerFunc {
	target, err := url.Parse(targetBase)
	if err != nil {
		panic(err)
	}

	return func(c *gin.Context) {
		// 读取 body（给实验窗口记录器用），再回填给代理请求
		var raw []byte
		if c.Request.Body != nil {
			raw, _ = io.ReadAll(c.Request.Body)
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(raw))

		// 从 /prod-api/{tag}/*path 提取真实路径
		path := c.Param("path")
		if path == "" {
			path = "/"
		}

		if opts.OnBeforeProxy != nil {
			opts.OnBeforeProxy(&ProxyContext{
				Method:  c.Request.Method,
				Path:    path,
				RawBody: raw,
			})
		}

		director := func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.URL.Path = singleJoiningSlash(target.Path, path)
			req.Host = target.Host

			// 透传 query
			req.URL.RawQuery = c.Request.URL.RawQuery

			// 回填 body
			req.Body = io.NopCloser(bytes.NewReader(raw))
			req.ContentLength = int64(len(raw))
		}

		proxy := &httputil.ReverseProxy{
			Director: director,
			ModifyResponse: func(resp *http.Response) error {
				if opts.LocationPrefix != "" {
					loc := resp.Header.Get("Location")
					if loc != "" {
						// 1) 绝对 URL：例如 http://localhost:3000/grafana/
						if u, err := url.Parse(loc); err == nil && u.IsAbs() {
							// 只改 path/query，最终回写成相对路径，避免 host 被污染
							newPath := singleJoiningSlash(opts.LocationPrefix, u.Path)
							if u.RawQuery != "" {
								resp.Header.Set("Location", newPath+"?"+u.RawQuery)
							} else {
								resp.Header.Set("Location", newPath)
							}
						} else if strings.HasPrefix(loc, "/") {
							// 2) 相对路径：例如 /grafana/ 或 /query?
							resp.Header.Set("Location", singleJoiningSlash(opts.LocationPrefix, loc))
						}
					}
				}

				if !opts.WrapResponse {
					return nil
				}

				ct := resp.Header.Get("Content-Type")
				// 仅包装 JSON；文件下载/二进制不包装
				if !strings.Contains(strings.ToLower(ct), "application/json") {
					return nil
				}

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					return err
				}
				_ = resp.Body.Close()

				// 尝试解析下游 JSON
				var payload any
				if len(body) > 0 {
					if err := json.Unmarshal(body, &payload); err != nil {
						// 非标准 JSON 也兜底包一下
						payload = string(body)
					}
				}

				// 4xx/5xx 返回 fail，其余 success
				var wrapped []byte
				if resp.StatusCode >= 400 {
					msg := http.StatusText(resp.StatusCode)
					// 尝试从下游 message/error 取更具体信息
					if m, ok := pickMessage(payload); ok {
						msg = m
					}
					wrapped, _ = json.Marshal(response.Fail(msg))
					resp.StatusCode = 200
				} else {
					wrapped, _ = json.Marshal(response.Success(payload))
					resp.StatusCode = 200
				}

				resp.Body = io.NopCloser(bytes.NewReader(wrapped))
				resp.ContentLength = int64(len(wrapped))
				resp.Header.Set("Content-Length", strconv.Itoa(len(wrapped)))
				resp.Header.Set("Content-Type", "application/json; charset=utf-8")
				return nil
			},
		}

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func pickMessage(v any) (string, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return "", false
	}
	if x, ok := m["message"].(string); ok && x != "" {
		return x, true
	}
	if x, ok := m["error"].(string); ok && x != "" {
		return x, true
	}
	if x, ok := m["msg"].(string); ok && x != "" {
		return x, true
	}
	return "", false
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}
