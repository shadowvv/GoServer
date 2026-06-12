package backend

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/backendPlatform"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/logic/webProto"
	"github.com/drop/GoServer/server/tool"
)

func returnSendMsg(w http.ResponseWriter, resp interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
	return
}

func sendErrorMessage(errorCode pb.ERROR_CODE, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(&webProto.WebErrorMessage{
		Code: int32(errorCode),
	})
}

func RegisterBackendMessage() {
	backendPlatform.RegisterHttpMessage("/manage/login", handleGmLogin)
	backendPlatform.RegisterHttpMessage("/manage/gmuser", handleGmUser)
	backendPlatform.RegisterHttpMessage("/manage/editgmuser", handleGmEditGmUser)
	backendPlatform.RegisterHttpMessage("/manage/userInfo", handleGmUserInfo)
	backendPlatform.RegisterHttpMessage("/manage/getformation", handleGmGetFormaiton)
	backendPlatform.RegisterHttpMessage("/manage/getaccessory", handleGmGetAccessory)
	backendPlatform.RegisterHttpMessage("/manage/usermail", handleGmUserMail)
	backendPlatform.RegisterHttpMessage("/manage/sendmail", handleGmSendMail)
	backendPlatform.RegisterHttpMessage("/manage/serverlist", handleGmServerList)
	backendPlatform.RegisterHttpMessage("/manage/editserver", handleGmEditServer)
	backendPlatform.RegisterHttpMessage("/manage/useritemchg", handleGmUserItemChg)
	backendPlatform.RegisterHttpMessage("/manage/userorder", handleGmUserOrder)
	backendPlatform.RegisterHttpMessage("/manage/getRankList", handleGmGetRankList) // 新增：获取排行榜列表
	backendPlatform.RegisterHttpMessage("/manage/getRank", handleGmGetRank)         // 获取具体排行榜数据
	backendPlatform.RegisterHttpMessage("/manage/gamepubilc", handleGmGamePublic)
	backendPlatform.RegisterHttpMessage("/manage/editgamepubilc", handleGmEditGamePublic)
	backendPlatform.RegisterHttpMessage("/manage/getTalk", handleGmGetTalk)
	// 编辑客户端版本，支持 JSON 和 multipart/form-data（上传配置文件），放宽 Body 上限到 50MB
	backendPlatform.RegisterHttpMessageWithMaxBody("/manage/editclientversion", handleGmEditClientVersion, 50<<20)
	backendPlatform.RegisterHttpMessage("/manage/getclientversion", handleGmGetClientVersion)
	backendPlatform.RegisterHttpMessage("/manage/getuserinventory", handleGmGetUserInventory)
	backendPlatform.RegisterHttpMessage("/manage/getServerActivityConfig", handleGmGetServerActivityConfig)
	backendPlatform.RegisterHttpMessage("/manage/editServerActivityConfig", handleGmEditServerActivityConfig)
	backendPlatform.RegisterHttpMessage("/manage/editBanUser", handleGmEditBanUser)
	backendPlatform.RegisterHttpMessage("/manage/editMuteUser", handleGmEditUserChat)
	backendPlatform.RegisterHttpMessage("/manage/getUserLogList", handleGmGetUserLogList)
	backendPlatform.RegisterHttpMessage("/manage/exportPlayer", handleGmExportPlayer)
	// 导入玩家数据 JSON 体积较大（可能数十 MB），单独放宽 Body 上限到 100MB
	backendPlatform.RegisterHttpMessageWithMaxBody("/manage/importPlayer", handleGmImportPlayer, 100<<20)
	backendPlatform.RegisterHttpMessage("/manage/getThroughput", handleGmGetThroughput)
	backendPlatform.RegisterHttpMessage("/manage/getServerActivityOpen", handleGmGetServerActivityOpen)
	backendPlatform.RegisterHttpMessage("/manage/editGmBuFa", handleEditGmBuFa)
	backendPlatform.RegisterHttpMessage("/manage/GmKickPlayer", handleGmKickPlayer)
}

func handleGmLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req GmLoginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}
	resp.Data = &GmLoginData{}
	respData := resp.Data.(*GmLoginData)

	//具体逻辑
	signStr := md5.Sum([]byte(req.PassWord))
	tMd5Pwd := fmt.Sprintf("%x", signStr)

	data := easyDB.GmGetEntityByWhere(enum.SystemUser, map[string]interface{}{"Account": req.UserName})

	if data == nil {
		resp.Code = 1
		resp.Msg = "user not found"
		returnSendMsg(w, resp)
		return
	}
	if data["PassWord"] == nil {
		resp.Code = 2
		resp.Msg = "password error"
		returnSendMsg(w, resp)
		return
	}
	if data["PassWord"] != tMd5Pwd {
		resp.Code = 2
		resp.Msg = "password error"
		returnSendMsg(w, resp)
		return
	}
	permiss := fmt.Sprintf("%v", data["Permiss"])
	respData.Permiss = gameConfig.ParseIntArray(permiss)
	respData.Token, _ = tool.GenToken(req.UserName, respData.Permiss)
	returnSendMsg(w, resp)
}

func handleGmUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req GmUserReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}
	resp.Data = make([]*GmUserData, 0)

	//具体逻辑
	tokenData, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		returnSendMsg(w, resp)
		return
	}
	if !enum.IsPermiss(tokenData.Permiss, enum.GmPermission) {
		resp.Code = -2002
		resp.Msg = "permiss is not exist"
		returnSendMsg(w, resp)
		return
	}

	GmUser(&req, resp)

	returnSendMsg(w, resp)
}

func handleGmEditGmUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req GmEditGmUserReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}

	//具体逻辑
	tokenData, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		returnSendMsg(w, resp)
		return
	}
	if !enum.IsPermiss(tokenData.Permiss, enum.GmPermission) {
		resp.Code = -2002
		resp.Msg = "permiss is not exist"
		returnSendMsg(w, resp)
		return
	}

	EditGmUser(&req, resp)

	returnSendMsg(w, resp)
}

func handleGmUserInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req GmUserInfoReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}
	resp.Data = make([]*GmUserInfoData, 0)

	//具体逻辑
	_, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		returnSendMsg(w, resp)
		return
	}

	GmUserInfo(&req, resp)

	returnSendMsg(w, resp)
}

func handleGmGetFormaiton(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req GmGetFormationReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}
	resp.Data = &GmGetFormationData{}

	//具体逻辑
	_, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		returnSendMsg(w, resp)
		return
	}

	GmGetFormation(&req, resp)

	returnSendMsg(w, resp)
}

func handleGmGetAccessory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req GmGetAccessoryReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}
	resp.Data = &GmGetAccessoryData{}

	//具体逻辑
	_, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		returnSendMsg(w, resp)
		return
	}

	GmGetAccessory(&req, resp)

	returnSendMsg(w, resp)
}

func handleGmUserMail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req GmUserMailReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}
	resp.Data = make([]*GmUserMailData, 0)

	//具体逻辑
	_, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		returnSendMsg(w, resp)
		return
	}
	GmUserMail(&req, resp)
	returnSendMsg(w, resp)
}

func handleGmSendMail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req GmSendMailReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}
	resp.Data = make([]*GmSendMailData, 0)

	//具体逻辑
	tokenData, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		returnSendMsg(w, resp)
		return
	}
	if !enum.IsPermiss(tokenData.Permiss, enum.SendMailPermission) {
		resp.Code = -2002
		resp.Msg = "permiss is not exist"
		returnSendMsg(w, resp)
		return
	}
	// 2. 编码为JSON
	jsonData, _ := json.Marshal(req)

	// 3. 发送POST请求
	backendResp, err := http.Post("http://127.0.0.1:8081/GmSendMail", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		resp.Code = -1
		resp.Msg = fmt.Sprintf("edit is error! err=%s", err.Error())
		returnSendMsg(w, resp)
		return
	}
	defer backendResp.Body.Close()

	var backendResult map[string]interface{}
	if err := json.NewDecoder(backendResp.Body).Decode(&backendResult); err != nil {
		resp.Code = -2
		resp.Msg = fmt.Sprintf("edit is error! err=%s", err.Error())
		returnSendMsg(w, resp)
		return
	}

	// 构造成功响应
	resp.Data = backendResult
	returnSendMsg(w, resp)
}

func handleGmServerList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req GmServerListReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}
	resp.Data = make([]*GmServerListData, 0)

	//具体逻辑
	_, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		returnSendMsg(w, resp)
		return
	}

	GmServerList(&req, resp)

	returnSendMsg(w, resp)
}

func handleGmEditServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req GmEditServerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}

	//具体逻辑
	if req.Info.ServerId < 0 {
		resp.Code = -1001
		resp.Msg = "serverId is error!"
		returnSendMsg(w, resp)
		return
	}
	tokenData, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		returnSendMsg(w, resp)
		return
	}
	if !enum.IsPermiss(tokenData.Permiss, enum.EditServerPermission) {
		resp.Code = -2002
		resp.Msg = "permiss is not exist"
		returnSendMsg(w, resp)
		return
	}

	// 2. 编码为JSON
	jsonData, _ := json.Marshal(req)

	// 3. 发送POST请求
	backendResp, err := http.Post("http://127.0.0.1:8081/GmEditServer", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		resp.Code = -1
		resp.Msg = fmt.Sprintf("edit is error! err=%s", err.Error())
		returnSendMsg(w, resp)
		return
	}
	defer backendResp.Body.Close()

	var backendResult map[string]interface{}
	if err := json.NewDecoder(backendResp.Body).Decode(&backendResult); err != nil {
		resp.Code = -2
		resp.Msg = fmt.Sprintf("edit is error! err=%s", err.Error())
		returnSendMsg(w, resp)
		return
	}

	// 构造成功响应
	resp.Data = backendResult
	returnSendMsg(w, resp)
}

func handleGmUserItemChg(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req GmUserItemChgReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}
	resp.Data = make([]*GmUserItemData, 0)

	//具体逻辑
	_, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		returnSendMsg(w, resp)
		return
	}

	GmUserItemChg(&req, resp)

	returnSendMsg(w, resp)
}

func handleGmUserOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req GmUserOrderReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}
	resp.Data = make([]*GmUserOrderData, 0)

	//具体逻辑
	_, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		returnSendMsg(w, resp)
		return
	}

	GmUserOrder(&req, resp)

	returnSendMsg(w, resp)
}

func handleGmGetRankList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req GmGetRankListReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}
	resp.Data = make([]*GmGetRankListData, 0)

	//具体逻辑
	_, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		returnSendMsg(w, resp)
		return
	}

	GmGetRankList(&req, resp)

	returnSendMsg(w, resp)
}

func handleGmGetRank(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req GmGetRankReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}
	resp.Data = make([]*GmGetRankData, 0)

	//具体逻辑
	_, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		returnSendMsg(w, resp)
		return
	}

	GmGetRank(&req, resp)

	returnSendMsg(w, resp)

}

func handleGmGamePublic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req GmGamePublicReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}
	resp.Data = make([]*GmGamePublicData, 0)

	//具体逻辑
	_, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		returnSendMsg(w, resp)
		return
	}

	GmGamePublic(&req, resp) // 查询游戏内公告
	returnSendMsg(w, resp)
}

// 游戏内公告
func handleGmEditGamePublic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req GmEditGamePublicReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}
	resp.Data = &GmGamePublicData{}

	//具体逻辑
	tokenData, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		returnSendMsg(w, resp)
		return
	}
	if !enum.IsPermiss(tokenData.Permiss, enum.EditGamePublicPermission) {
		resp.Code = -2002
		resp.Msg = "permiss is not exist"
		returnSendMsg(w, resp)
		return
	}

	// 2. 编码为 JSON
	jsonData, _ := json.Marshal(req)

	// 3. 发送 POST 请求到 game server
	backendResp, err := http.Post("http://127.0.0.1:8081/GmEditGamePublic", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		resp.Code = -1
		resp.Msg = fmt.Sprintf("edit is error! err=%s", err.Error())
		returnSendMsg(w, resp)
		return
	}
	defer backendResp.Body.Close()

	var backendResult *GmResp
	if err := json.NewDecoder(backendResp.Body).Decode(&backendResult); err != nil {
		resp.Code = -2
		resp.Msg = fmt.Sprintf("edit is error! err=%s", err.Error())
		returnSendMsg(w, resp)
		return
	}

	// 构造成功响应
	resp = backendResult
	returnSendMsg(w, resp)
}

func handleGmGetTalk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req GmGetTalkReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}
	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}
	resp.Data = &webProto.GetChatMessageResp{}

	//具体逻辑
	_, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		returnSendMsg(w, resp)
		return
	}

	GmGetTalk(&req, resp)

	returnSendMsg(w, resp)
}

func handleGmEditClientVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	// 判断是否是上传文件请求（multipart/form-data）
	if r.Header.Get("Content-Type") != "" && len(r.Header.Get("Content-Type")) > 19 && r.Header.Get("Content-Type")[:19] == "multipart/form-data" {
		// 转发 multipart 请求到 game server
		if err := r.ParseMultipartForm(50 << 20); err != nil {
			resp := &GmResp{Code: -1, Msg: fmt.Sprintf("parse form error: %s", err.Error())}
			returnSendMsg(w, resp)
			return
		}

		// 创建新的 multipart 请求转发
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// 复制表单字段
		for key, values := range r.MultipartForm.Value {
			for _, value := range values {
				writer.WriteField(key, value)
			}
		}

		// 复制文件
		if fileHeaders, ok := r.MultipartForm.File["file"]; ok {
			for _, fileHeader := range fileHeaders {
				file, err := fileHeader.Open()
				if err != nil {
					resp := &GmResp{Code: -1, Msg: fmt.Sprintf("open file error: %s", err.Error())}
					returnSendMsg(w, resp)
					return
				}
				defer file.Close()

				part, err := writer.CreateFormFile("file", fileHeader.Filename)
				if err != nil {
					resp := &GmResp{Code: -1, Msg: fmt.Sprintf("create form file error: %s", err.Error())}
					returnSendMsg(w, resp)
					return
				}
				io.Copy(part, file)
			}
		}
		writer.Close()

		// 发送 POST 请求到 game server
		req, err := http.NewRequest("POST", "http://127.0.0.1:8081/GmEditClientVersion", body)
		if err != nil {
			resp := &GmResp{Code: -1, Msg: fmt.Sprintf("create request error: %s", err.Error())}
			returnSendMsg(w, resp)
			return
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())

		backendResp, err := http.DefaultClient.Do(req)
		if err != nil {
			resp := &GmResp{Code: -1, Msg: fmt.Sprintf("forward request error: %s", err.Error())}
			returnSendMsg(w, resp)
			return
		}
		defer backendResp.Body.Close()

		var backendResult map[string]interface{}
		if err := json.NewDecoder(backendResp.Body).Decode(&backendResult); err != nil {
			resp := &GmResp{Code: -2, Msg: fmt.Sprintf("decode response error: %s", err.Error())}
			returnSendMsg(w, resp)
			return
		}

		resp := &GmResp{Code: 0, Msg: "", Data: backendResult}
		returnSendMsg(w, resp)
		return
	}

	// 原有的 JSON 请求逻辑
	var req GmEditClientVersionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}

	//具体逻辑
	tokenData, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		returnSendMsg(w, resp)
		return
	}
	if !enum.IsPermiss(tokenData.Permiss, enum.EditClientVersionPermission) {
		resp.Code = -2002
		resp.Msg = "permiss is not exist"
		returnSendMsg(w, resp)
		return
	}

	// 2. 编码为JSON
	jsonData, _ := json.Marshal(req)

	// 3. 发送POST请求
	backendResp, err := http.Post("http://127.0.0.1:8081/GmEditClientVersion", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		resp.Code = -1
		resp.Msg = fmt.Sprintf("edit is error! err=%s", err.Error())
		returnSendMsg(w, resp)
		return
	}
	defer backendResp.Body.Close()

	var backendResult map[string]interface{}
	if err := json.NewDecoder(backendResp.Body).Decode(&backendResult); err != nil {
		resp.Code = -2
		resp.Msg = fmt.Sprintf("edit is error! err=%s", err.Error())
		returnSendMsg(w, resp)
		return
	}

	// 构造成功响应
	resp.Data = backendResult
	returnSendMsg(w, resp)
}

func handleGmGetClientVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req GmGetClientVersionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}
	resp.Data = make([]*GmClientVersionData, 0)

	//具体逻辑
	_, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		returnSendMsg(w, resp)
		return
	}

	GmGetClientVersion(&req, resp)

	returnSendMsg(w, resp)
}

func handleGmGetUserInventory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req GmGetUserInventoryReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}
	resp.Data = make([]*GmInventoryData, 0)

	//具体逻辑
	_, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		returnSendMsg(w, resp)
		return
	}

	GmGetUserInventory(&req, resp)

	returnSendMsg(w, resp)
}

func handleGmGetServerActivityConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req GmGetServerActivityConfigReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}
	resp.Data = make([]*GmServerActivityConfigData, 0)

	_, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		returnSendMsg(w, resp)
		return
	}

	GmGetServerActivityConfig(&req, resp)

	returnSendMsg(w, resp)
}

func handleGmEditServerActivityConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req GmEditServerActivityConfigReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}

	tokenData, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		returnSendMsg(w, resp)
		return
	}
	if !enum.IsPermiss(tokenData.Permiss, enum.EditActivityPermission) {
		resp.Code = -2002
		resp.Msg = "permiss is not exist"
		returnSendMsg(w, resp)
		return
	}

	jsonData, _ := json.Marshal(req)

	backendResp, err := http.Post("http://127.0.0.1:8081/GmEditServerActivityConfig", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		resp.Code = -1
		resp.Msg = fmt.Sprintf("edit is error! err=%s", err.Error())
		returnSendMsg(w, resp)
		return
	}
	defer backendResp.Body.Close()

	var backendResult GmResp
	if err := json.NewDecoder(backendResp.Body).Decode(&backendResult); err != nil {
		resp.Code = -2
		resp.Msg = fmt.Sprintf("edit is error! err=%s", err.Error())
		returnSendMsg(w, resp)
		return
	}

	resp.Code = backendResult.Code
	resp.Msg = backendResult.Msg
	resp.Data = backendResult.Data
	returnSendMsg(w, resp)
}

func handleGmEditBanUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req GmEditBanUserReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}
	resp.Data = &GmBanUserData{}

	tokenData, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		returnSendMsg(w, resp)
		return
	}
	if !enum.IsPermiss(tokenData.Permiss, enum.EditBanUserPermission) {
		resp.Code = -2002
		resp.Msg = "permiss is not exist"
		returnSendMsg(w, resp)
		return
	}

	GmEditBanUser(&req, resp)

	returnSendMsg(w, resp)
}

func handleGmEditUserChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req GmEditUserChatReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}
	resp.Data = &GmEditUserChatData{}

	tokenData, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		returnSendMsg(w, resp)
		return
	}
	if !enum.IsPermiss(tokenData.Permiss, enum.EditMuteUserPermission) {
		resp.Code = -2002
		resp.Msg = "permiss is not exist"
		returnSendMsg(w, resp)
		return
	}

	GmEditUserChat(&req, resp)

	returnSendMsg(w, resp)
}

func handleGmGetUserLogList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req GmGetUserLogListReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}
	resp.Data = make([]*GmUserLogData, 0)

	_, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		returnSendMsg(w, resp)
		return
	}

	GmGetUserLogList(&req, resp)

	returnSendMsg(w, resp)
}

func handleGmExportPlayer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	// 延长写超时，防止 SQL 生成耗时过长导致连接断开
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Now().Add(10 * time.Minute))

	var req GmExportPlayerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	tokenData, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp := &GmResp{Code: -2001, Msg: fmt.Sprintf("token is error! err2=%s", err2.Error())}
		returnSendMsg(w, resp)
		return
	}
	if !enum.IsPermiss(tokenData.Permiss, enum.ExportPlayerPermission) {
		resp := &GmResp{Code: -2002, Msg: "permiss is not exist"}
		returnSendMsg(w, resp)
		return
	}

	resp := &GmResp{Code: 0, Msg: ""}
	resp.Data = &GmExportPlayerData{}
	GmExportPlayer(&req, resp)

	if resp.Code != 0 {
		returnSendMsg(w, resp)
		return
	}

	// 成功：直接流式分块返回 JSON 内容，不落盘
	respData := resp.Data.(*GmExportPlayerData)
	jsonBytes := []byte(respData.Json)
	respData.Json = "" // 释放原始字符串内存

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%d_player.json", req.UserId))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(jsonBytes)))

	// 分块写入 + 每块重置写超时，确保大数据传输不超时
	const chunkSize = 64 * 1024 // 64KB per chunk
	for offset := 0; offset < len(jsonBytes); {
		_ = rc.SetWriteDeadline(time.Now().Add(60 * time.Second))
		end := offset + chunkSize
		if end > len(jsonBytes) {
			end = len(jsonBytes)
		}
		if _, err := w.Write(jsonBytes[offset:end]); err != nil {
			return
		}
		_ = rc.Flush()
		offset = end
	}
}

func handleGmImportPlayer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req GmImportPlayerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}
	resp.Data = &GmImportPlayerData{}

	tokenData, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		returnSendMsg(w, resp)
		return
	}
	if !enum.IsPermiss(tokenData.Permiss, enum.ImportPlayerPermission) {
		resp.Code = -2002
		resp.Msg = "permiss is not exist"
		returnSendMsg(w, resp)
		return
	}

	GmImportPlayer(&req, resp)

	returnSendMsg(w, resp)
}

func handleGmGetThroughput(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req GmGetThroughputReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}
	resp.Data = make([]*GmThroughputItem, 0)

	_, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		returnSendMsg(w, resp)
		return
	}

	GmGetThroughput(&req, resp)

	returnSendMsg(w, resp)
}

func handleGmGetServerActivityOpen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req GmGetServerActivityOpenReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}
	resp.Data = &GmGetServerActivityOpenData{}

	_, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		returnSendMsg(w, resp)
		return
	}

	GmGetServerActivityOpen(&req, resp)

	returnSendMsg(w, resp)
}

func handleEditGmBuFa(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}
	var req EditGmBuFaReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}
	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}

	tokenData, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		returnSendMsg(w, resp)
		return
	}
	if !enum.IsPermiss(tokenData.Permiss, enum.BuFaPermission) {
		resp.Code = -2002
		resp.Msg = "permiss is not exist"
		returnSendMsg(w, resp)
		return
	}

	toReq := &webProto.ConsumeProductReq{
		PlayerId:  req.UserId,
		OrderInfo: []*webProto.OrderInfo{},
	}
	orderInfo := &webProto.OrderInfo{
		OrderId:   req.OrderId,
		ProductId: req.ProductId,
		Token:     "",
	}
	toReq.OrderInfo = append(toReq.OrderInfo, orderInfo)
	jsonData, _ := json.Marshal(toReq)

	backendResp, err := http.Post("http://127.0.0.1:8081/gmConsumeProduct", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		resp.Code = -1
		resp.Msg = fmt.Sprintf("edit is error! err=%s", err.Error())
		returnSendMsg(w, resp)
		return
	}
	defer backendResp.Body.Close()

	var backendResult map[string]interface{}
	if err := json.NewDecoder(backendResp.Body).Decode(&backendResult); err != nil {
		resp.Code = -2
		resp.Msg = fmt.Sprintf("edit is error! err=%s", err.Error())
		returnSendMsg(w, resp)
		return
	}

	resp.Data = backendResult
	returnSendMsg(w, resp)
}

func handleGmKickPlayer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}
	// 转发给http
	var req GmKickPlayerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}
	resp := &GmResp{
		Code: 0,
		Msg:  "",
	}

	tokenData, err2 := tool.ParseToken(req.Token)
	if err2 != nil {
		resp.Code = -2001
		resp.Msg = fmt.Sprintf("token is error! err2=%s", err2.Error())
		returnSendMsg(w, resp)
		return
	}
	if !enum.IsPermiss(tokenData.Permiss, enum.KickPlayerPermission) {
		resp.Code = -2002
		resp.Msg = "permiss is not exist"
		returnSendMsg(w, resp)
		return
	}
	jsonData, _ := json.Marshal(req)

	backendResp, err := http.Post("http://127.0.0.1:8081/GmKickPlayer", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		resp.Code = -1
		resp.Msg = fmt.Sprintf("edit is error! err=%s", err.Error())
		returnSendMsg(w, resp)
		return
	}
	defer backendResp.Body.Close()

	var backendResult map[string]interface{}
	if err := json.NewDecoder(backendResp.Body).Decode(&backendResult); err != nil {
		resp.Code = -2
		resp.Msg = fmt.Sprintf("edit is error! err=%s", err.Error())
		returnSendMsg(w, resp)
		return
	}

	resp.Data = backendResult
	returnSendMsg(w, resp)
}
