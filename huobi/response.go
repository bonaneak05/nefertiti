package huobi

import (
	"encoding/json"
)

type Response struct {
	Status string `json:"status"`
	ErrMsg string `json:"err-msg"`
}

func IsError(body []byte) (bool, string) {
	var resp Response
	if json.Unmarshal(body, &resp) == nil {
		return resp.Status != "ok", resp.ErrMsg
	}
	return false, ""
}
