package response

type APIResponse struct {
	Code string `json:"code"`
	Data any    `json:"data"`
	Msg  string `json:"msg"`
}

func Success(data any) APIResponse {
	return APIResponse{
		Code: "00000",
		Data: data,
		Msg:  "ok",
	}
}

func Fail(msg string) APIResponse {
	return APIResponse{
		Code: "B0001",
		Data: nil,
		Msg:  msg,
	}
}
