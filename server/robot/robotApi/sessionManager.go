package robotApi

import (
	"fmt"
	"sync"
	"time"

	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/robot/robotConfig"
	"github.com/drop/GoServer/server/robot/robotLogic"
	"github.com/drop/GoServer/server/robot/robotMonitor"
)

type Session struct {
	ID        string
	Account   string
	CreatedAt time.Time
	Robot     *robotLogic.Robot
}

type SessionManager struct {
	mu               sync.RWMutex
	cfg              *robotConfig.RobotConfig
	monitor          *robotMonitor.PlatformMonitor
	sessions         map[string]*Session
	defaultSessionID string
}

func NewSessionManager(cfg *robotConfig.RobotConfig) *SessionManager {
	return &SessionManager{
		cfg:      cfg,
		monitor:  robotMonitor.NewPlatformMonitor(),
		sessions: make(map[string]*Session),
	}
}

func (m *SessionManager) StartSession(req *StartSessionRequest) (*Session, error) {
	if m.cfg == nil {
		return nil, fmt.Errorf("robot config is nil")
	}

	login := m.cfg.Main.Login
	run := m.cfg.Main.RunConfig

	account := buildSessionAccount(login.Account, req.Account, time.Now().UnixNano())
	serverID := req.ServerID
	if serverID == 0 {
		serverID = login.ServerID
	}
	loginURL := req.LoginURL
	if loginURL == "" {
		loginURL = login.LoginURL
	}
	channel := req.Channel
	if channel == 0 {
		channel = login.Channel
	}
	version := req.Version
	if version == "" {
		version = login.Version
	}
	language := req.Language
	if language == 0 {
		language = login.Language
	}
	deviceID := req.DeviceID
	if deviceID == "" {
		deviceID = login.DeviceID
	}
	appID := req.AppID
	if appID == "" {
		appID = login.AppID
	}
	sign := req.Sign
	if sign == "" {
		sign = login.Sign
	}

	sessionID := fmt.Sprintf("%s_%d", account, time.Now().UnixNano())
	r := robotLogic.NewRobot(sessionID, account, serverID, loginURL, channel, version, language, deviceID, appID, sign, m.cfg, run.Mode, run.Interval, run.Duration, m.monitor)
	r.DisableAutoOperations()

	if err := r.Start(); err != nil {
		return nil, err
	}
	if !r.WaitReady(15 * time.Second) {
		r.Stop()
		return nil, fmt.Errorf("robot wait ready timeout")
	}

	session := &Session{
		ID:        sessionID,
		Account:   account,
		CreatedAt: time.Now(),
		Robot:     r,
	}
	m.registerSession(session)
	return session, nil
}

func (m *SessionManager) SendOperation(sessionID string, messageID pb.MESSAGE_ID, params map[string]interface{}) (*robotLogic.OperationSendResult, error) {
	session, ok := m.GetSession(sessionID)
	if !ok {
		if sessionID != "" {
			return nil, fmt.Errorf("session not found")
		}
		var err error
		session, err = m.StartSession(&StartSessionRequest{})
		if err != nil {
			return nil, err
		}
	}
	return session.Robot.SendOperation(messageID, params)
}

func (m *SessionManager) StopSession(sessionID string) bool {
	session, ok := m.GetSession(sessionID)
	if !ok {
		return false
	}
	if session.Robot != nil {
		session.Robot.Stop()
	}
	m.removeSession(session.ID)
	return true
}

func (m *SessionManager) ListSessions() []*SessionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]*SessionInfo, 0, len(m.sessions))
	for _, session := range m.sessions {
		ready := false
		if session.Robot != nil {
			ready = session.Robot.IsReady()
		}
		out = append(out, &SessionInfo{
			ID:        session.ID,
			Account:   session.Account,
			CreatedAt: session.CreatedAt.Unix(),
			Ready:     ready,
		})
	}
	return out
}

func (m *SessionManager) GetSession(sessionID string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if sessionID == "" {
		sessionID = m.defaultSessionID
	}
	session, ok := m.sessions[sessionID]
	return session, ok
}

func (m *SessionManager) registerSession(session *Session) {
	m.mu.Lock()
	m.sessions[session.ID] = session
	if m.defaultSessionID == "" {
		m.defaultSessionID = session.ID
	}
	m.mu.Unlock()
}

func (m *SessionManager) removeSession(sessionID string) {
	m.mu.Lock()
	delete(m.sessions, sessionID)
	if m.defaultSessionID == sessionID {
		m.defaultSessionID = ""
	}
	m.mu.Unlock()
}

func buildSessionAccount(baseAccount string, reqAccount string, now int64) string {
	if reqAccount != "" {
		return reqAccount
	}
	return fmt.Sprintf("%s_%d", baseAccount, now)
}
