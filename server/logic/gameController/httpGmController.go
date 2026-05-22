package gameController

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/backend"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/httpPlatform"
	"github.com/drop/GoServer/server/logic/rpcController"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/tool"
)

func RegisterWebGmMessage() {
	httpPlatform.RegisterHttpMessage("/GmEditServer", handleGmEditServer)
	httpPlatform.RegisterHttpMessage("/GmSendMail", handleGmSendMail)
	httpPlatform.RegisterHttpMessage("/GmEditGamePublic", handleGmEditGamePublic)
	httpPlatform.RegisterHttpMessage("/GmEditClientVersion", handleGmEditClientVersion)
	httpPlatform.RegisterHttpMessage("/GmGetClientVersion", handleGmGetClientVersion)
	httpPlatform.RegisterHttpMessage("/GmEditServerActivityConfig", handleGmEditServerActivityConfig)
	httpPlatform.RegisterHttpMessage("/GmKickPlayer", handleGmKickPlayer)
}

func handleGmEditServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req backend.GmEditServerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &backend.GmResp{
		Code: 0,
		Msg:  "",
	}

	backend.GmEditServer(&req, resp)

	rpcController.SendOperationToGateway(rpcPb.RPC_SERVER_OPERATION_RPC_OPERATION_RELOAD_SERVER_INFO)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func handleGmSendMail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req backend.GmSendMailReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &backend.GmResp{
		Code: 0,
		Msg:  "",
	}
	resp.Data = &backend.GmSendMailData{}

	backend.GmSendMail(&req, resp)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// 游戏内公告
func handleGmEditGamePublic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req backend.GmEditGamePublicReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &backend.GmResp{
		Code: 0,
		Msg:  "",
	}
	resp.Data = &backend.GmEditGamePublicData{}

	backend.GmEditGamePublic(&req, resp)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func handleGmEditClientVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req backend.GmEditClientVersionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &backend.GmResp{
		Code: 0,
		Msg:  "",
	}

	backend.GmEditClientVersion(&req, resp)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func handleGmGetClientVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req backend.GmGetClientVersionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &backend.GmResp{
		Code: 0,
		Msg:  "",
	}
	resp.Data = make([]*backend.GmClientVersionData, 0)

	backend.GmGetClientVersion(&req, resp)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func handleGmEditServerActivityConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req backend.GmEditServerActivityConfigReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &backend.GmResp{
		Code: 0,
		Msg:  "",
	}

	backend.GmEditServerActivityConfig(&req, resp)

	rpcController.BroadcastOperationToGameNode(rpcPb.RPC_SERVER_OPERATION_RPC_OPERATION_RELOAD_ACTIVITY_CONFIG)
	rpcController.SendOperationToGateway(rpcPb.RPC_SERVER_OPERATION_RPC_OPERATION_RELOAD_ACTIVITY_CONFIG)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// TODO:
func handleGmKickPlayer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req backend.GmKickPlayerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &backend.GmResp{
		Code: 0,
		Msg:  "",
	}

	tokenData, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}
	if !enum.IsPermiss(tokenData.Permiss, enum.KickPlayerPermission) {
		resp.Code = -2002
		resp.Msg = "permiss is not exist"
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	backend.GmKickPlayer(&req, resp)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
