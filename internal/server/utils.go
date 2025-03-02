package server

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func SuccessResp(w http.ResponseWriter, code int, payload interface{}) {
	resp, err := json.Marshal(payload)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(code)
	w.Write(resp)
	return
}

func ErrorResp(w http.ResponseWriter, code int, msg string) {
	type ErrResp struct {
		Error string `json:"error"`
	}
	err_resp := ErrResp{Error: msg}
	resp, err := json.Marshal(err_resp)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(code)
	w.Write(resp)
	return
}
