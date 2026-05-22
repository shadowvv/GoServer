package robotApi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/robot/robotConfig"
	_ "github.com/drop/GoServer/server/robot/robotModuleController"
	"github.com/drop/GoServer/server/robot/robotRouter"
	"github.com/drop/GoServer/server/robot/robotUtils"
	"github.com/drop/GoServer/server/service/logger"
)

type Server struct {
	address      string
	manager      *SessionManager
	directRoutes []directRoute
}

type route struct {
	path    string
	handler http.HandlerFunc
}

type directRoute struct {
	path      string
	messageID pb.MESSAGE_ID
}

func NewServer(address string) (*Server, error) {
	if err := robotRouter.RegisterAllRobotMessages(); err != nil {
		return nil, err
	}

	directRoutes, err := loadDirectRoutes("config/robotApiRoutes.yaml")
	if err != nil {
		return nil, err
	}

	cfg, err := robotConfig.LoadConfig("config/robot.yaml")
	if err != nil {
		return nil, err
	}
	if err = logger.InitLoggerByConfig(&cfg.Logger); err != nil {
		return nil, err
	}
	if err = cfg.BuildRealOperations(); err != nil {
		return nil, err
	}

	return &Server{
		address:      address,
		manager:      NewSessionManager(cfg),
		directRoutes: directRoutes,
	}, nil
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	for _, route := range s.routes() {
		mux.HandleFunc(route.path, route.handler)
	}
	for _, route := range s.directRoutes {
		mux.HandleFunc(route.path, s.handleDirectOperation(route.messageID))
	}

	logger.InfoWithSprintf("[robotApi] start listen: %s", s.address)
	return http.ListenAndServe(s.address, mux)
}

func (s *Server) routes() []route {
	return []route{
		{path: "/robot/session/start", handler: s.handleStartSession},
		{path: "/robot/session/send", handler: s.handleSendOperation},
		{path: "/robot/session/stop", handler: s.handleStopSession},
		{path: "/robot/session/list", handler: s.handleListSession},
		{path: "/robot/send", handler: s.handleSendOperation},
	}
}

func (s *Server) handleStartSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, pb.ERROR_CODE_INVALID_REQUEST_PARAM, "method must be POST")
		return
	}

	var req StartSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, pb.ERROR_CODE_INVALID_REQUEST_PARAM, err.Error())
		return
	}

	session, err := s.manager.StartSession(&req)
	if err != nil {
		writeError(w, pb.ERROR_CODE_SYSTEM_ERROR, err.Error())
		return
	}

	writeOK(w, &SessionInfo{
		ID:        session.ID,
		Account:   session.Account,
		CreatedAt: session.CreatedAt.Unix(),
		Ready:     session.Robot != nil && session.Robot.IsReady(),
	})
}

func (s *Server) handleSendOperation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, pb.ERROR_CODE_INVALID_REQUEST_PARAM, "method must be POST")
		return
	}

	var req SendOperationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, pb.ERROR_CODE_INVALID_REQUEST_PARAM, err.Error())
		return
	}
	if req.MessageID == "" {
		writeError(w, pb.ERROR_CODE_INVALID_REQUEST_PARAM, "messageId is required")
		return
	}

	messageID, err := robotUtils.ParseMessageID(req.MessageID)
	if err != nil {
		writeError(w, pb.ERROR_CODE_INVALID_REQUEST_PARAM, err.Error())
		return
	}

	result, err := s.manager.SendOperation(req.SessionID, messageID, req.Params)
	if err != nil {
		writeError(w, pb.ERROR_CODE_SYSTEM_ERROR, err.Error())
		return
	}
	writeOK(w, result)
}

func (s *Server) handleDirectOperation(messageID pb.MESSAGE_ID) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, pb.ERROR_CODE_INVALID_REQUEST_PARAM, "method must be POST")
			return
		}

		req, err := decodeDirectOperationRequest(r.Body)
		if err != nil {
			writeError(w, pb.ERROR_CODE_INVALID_REQUEST_PARAM, err.Error())
			return
		}

		result, err := s.manager.SendOperation(req.SessionID, messageID, req.Params)
		if err != nil {
			writeError(w, pb.ERROR_CODE_SYSTEM_ERROR, err.Error())
			return
		}
		writeOK(w, result)
	}
}

func (s *Server) handleStopSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, pb.ERROR_CODE_INVALID_REQUEST_PARAM, "method must be POST")
		return
	}

	var req StopSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, pb.ERROR_CODE_INVALID_REQUEST_PARAM, err.Error())
		return
	}
	sessionID := req.SessionID
	if sessionID == "" {
		if session, ok := s.manager.GetSession(""); ok {
			sessionID = session.ID
		}
	}
	if !s.manager.StopSession(req.SessionID) {
		writeError(w, pb.ERROR_CODE_INVALID_REQUEST_PARAM, "session not found")
		return
	}
	writeOK(w, map[string]string{"sessionId": sessionID})
}

func (s *Server) handleListSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, pb.ERROR_CODE_INVALID_REQUEST_PARAM, "method must be GET")
		return
	}
	writeOK(w, s.manager.ListSessions())
}

func decodeDirectOperationRequest(body io.Reader) (*DirectOperationRequest, error) {
	raw := make(map[string]interface{})
	decoder := json.NewDecoder(body)
	if err := decoder.Decode(&raw); err != nil && err != io.EOF {
		return nil, err
	}

	req := &DirectOperationRequest{}
	if sessionID, ok := raw["sessionId"].(string); ok {
		req.SessionID = sessionID
	}

	if params, ok := raw["params"]; ok {
		if params == nil {
			return req, nil
		}
		paramMap, ok := params.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("params must be object")
		}
		req.Params = paramMap
		return req, nil
	}

	delete(raw, "sessionId")
	if len(raw) > 0 {
		req.Params = raw
	}
	return req, nil
}

func writeOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(&Response{
		Code: int32(pb.ERROR_CODE_SUCCESS),
		Data: data,
	})
}

func writeError(w http.ResponseWriter, code pb.ERROR_CODE, message string) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(&Response{
		Code:    int32(code),
		Message: fmt.Sprintf("%s", message),
	})
}
