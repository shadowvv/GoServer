package robotApi

type StartSessionRequest struct {
	Account  string `json:"account"`
	ServerID int32  `json:"serverId"`
	LoginURL string `json:"loginUrl"`
	Channel  int32  `json:"channel"`
	Version  string `json:"version"`
	Language uint16 `json:"language"`
	DeviceID string `json:"deviceId"`
	AppID    string `json:"appId"`
	Sign     string `json:"sign"`
}

type SendOperationRequest struct {
	SessionID string                 `json:"sessionId"`
	MessageID string                 `json:"messageId"`
	Params    map[string]interface{} `json:"params"`
}

type DirectOperationRequest struct {
	SessionID string                 `json:"sessionId"`
	Params    map[string]interface{} `json:"params"`
}

type StopSessionRequest struct {
	SessionID string `json:"sessionId"`
}

type Response struct {
	Code    int32       `json:"code"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type SessionInfo struct {
	ID        string `json:"sessionId"`
	Account   string `json:"account"`
	CreatedAt int64  `json:"createdAt"`
	Ready     bool   `json:"ready"`
}
