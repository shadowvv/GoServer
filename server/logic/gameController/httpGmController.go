package gameController

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/backend"
	"github.com/drop/GoServer/server/logic/gameConfig"
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

	rpcController.SendOperationToGateway(rpcPb.RPC_SERVER_OPERATION_RPC_OPERATION_RELOAD_SERVER_INFO, 0)

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

	resp := &backend.GmResp{
		Code: 0,
		Msg:  "",
	}

	// 判断是否是上传文件请求（multipart/form-data）
	if r.Header.Get("Content-Type") != "" && len(r.Header.Get("Content-Type")) > 19 && r.Header.Get("Content-Type")[:19] == "multipart/form-data" {
		// 解析表单
		if err := r.ParseMultipartForm(50 << 20); err != nil {
			resp.Code = -1
			resp.Msg = fmt.Sprintf("parse form error: %s", err.Error())
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		token := r.FormValue("token")
		version := r.FormValue("version")
		if token == "" || version == "" {
			resp.Code = -1
			resp.Msg = "token and version are required"
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// 验证 token 和权限
		tokenData, err2 := tool.ParseToken(token)
		if err2 != nil {
			resp.Code = -2001
			resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		if !enum.IsPermiss(tokenData.Permiss, enum.EditClientVersionPermission) {
			resp.Code = -2002
			resp.Msg = "permiss is not exist"
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// 获取上传的文件
		file, fileHeader, err := r.FormFile("file")
		if err != nil {
			resp.Code = -1
			resp.Msg = fmt.Sprintf("get file error: %s", err.Error())
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		defer file.Close()

		// 构造请求，把文件赋值给 UploadFile
		req := backend.GmEditClientVersionReq{
			Token: token,
			ClientVersionList: &backend.GmClientVersionData{
				Version:    version,
				UploadFile: fileHeader,
			},
		}

		backend.GmEditClientVersion(&req, resp)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	// 原有的 JSON 请求逻辑
	var req backend.GmEditClientVersionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
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

	if req.Data == nil {
		resp.Code = 1
		resp.Msg = "invalid request: data is nil"
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}
	if err := gameConfig.CheckActivityConfig(&gameConfig.ActivityConfigCheckData{
		Id:              req.Data.Id,
		ServerType:      req.Data.ServerType,
		ServerUnit:      req.Data.ServerUnit,
		UnlockIds:       gameConfig.ParseIntArray(req.Data.UnlockId),
		AttendUnlockIds: gameConfig.ParseIntArray(req.Data.AttendUnlockId),
		EventOpen:       req.Data.EventOpen,
		EventEnd:        req.Data.EventEnd,
		WeekOpenDays:    gameConfig.ParseIntArray(req.Data.WeekOpen),
		MonthOpenDays:   gameConfig.ParseIntArray(req.Data.MonthOpen),
		Duration:        req.Data.Duration,
		NextId:          req.Data.NextId,
		OpenLoopMax:     req.Data.OpenLoopNum,
	}); err != nil {
		resp.Code = 1
		resp.Msg = fmt.Sprintf("invalid activity config: %v", err)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	backend.GmEditServerActivityConfig(&req, resp)

	if resp.Code == 0 {
		rpcController.SendOperationToGateway(rpcPb.RPC_SERVER_OPERATION_RPC_OPERATION_RELOAD_ACTIVITY_CONFIG, 0)
	}

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
