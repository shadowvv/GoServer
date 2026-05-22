package loginMutexService

import (
	"fmt"
	"github.com/drop/GoServer/server/service/logger"
	"sync"
)

var loginMutexMap = &LoginMutexMap{
	accountMap: make(map[string]bool),
	sessionMap: make(map[int64]bool),
}

func EnterMutex(account string, sessionId int64) bool {
	return loginMutexMap.enterMutex(account, sessionId)
}

func EnterAccountMutex(account string) bool {
	return loginMutexMap.enterAccountMutex(account)
}

func ExitMutex(account string, sessionId int64) {
	loginMutexMap.exitMutex(account, sessionId)
}

func ExitAccountMutex(account string) {
	loginMutexMap.exitAccountMutex(account)
}

func EnterSessionMutex(sessionId int64) bool {
	return loginMutexMap.EnterSessionMutex(sessionId)
}

func ExitSessionMutex(sessionId int64) {
	loginMutexMap.ExitSessionMutex(sessionId)
}

// 登录锁
type LoginMutexMap struct {
	sync.Mutex
	accountMap map[string]bool // account -> true
	sessionMap map[int64]bool  // sessionId -> true
}

func (l *LoginMutexMap) enterMutex(account string, sessionId int64) bool {
	l.Lock()
	defer l.Unlock()
	logger.Warn(fmt.Sprintf("try enter mutex account:%s,sessionId:%d", account, sessionId))
	if _, ok := l.accountMap[account]; ok {
		return false
	}
	if _, ok := l.sessionMap[sessionId]; ok {
		return false
	}
	l.accountMap[account] = true
	l.sessionMap[sessionId] = true
	logger.Warn(fmt.Sprintf("enter mutex account:%s,sessionId:%d", account, sessionId))
	return true
}

func (l *LoginMutexMap) enterAccountMutex(account string) bool {
	l.Lock()
	defer l.Unlock()
	logger.Warn(fmt.Sprintf("try enter account mutex account:%s", account))
	if _, ok := l.accountMap[account]; ok {
		return false
	}
	l.accountMap[account] = true
	logger.Warn(fmt.Sprintf("enter account mutex account:%s", account))
	return true
}

func (l *LoginMutexMap) exitMutex(account string, sessionId int64) {
	l.Lock()
	defer l.Unlock()
	logger.Warn(fmt.Sprintf("try exit mutex account:%s,sessionId:%d", account, sessionId))
	delete(l.accountMap, account)
	delete(l.sessionMap, sessionId)
	logger.Warn(fmt.Sprintf("exit mutex account:%s,sessionId:%d", account, sessionId))
}

func (l *LoginMutexMap) exitAccountMutex(account string) {
	l.Lock()
	defer l.Unlock()
	logger.Warn(fmt.Sprintf("try exit account mutex account:%s", account))
	delete(l.accountMap, account)
	logger.Warn(fmt.Sprintf("exit account mutex account:%s", account))
}

func (l *LoginMutexMap) EnterSessionMutex(sessionId int64) bool {
	l.Lock()
	defer l.Unlock()
	logger.Warn(fmt.Sprintf("try enter session mutex sessionId:%d", sessionId))
	if _, ok := l.sessionMap[sessionId]; ok {
		return false
	}
	l.sessionMap[sessionId] = true
	logger.Warn(fmt.Sprintf("enter session mutex sessionId:%d", sessionId))
	return true
}

func (l *LoginMutexMap) ExitSessionMutex(sessionId int64) {
	l.Lock()
	defer l.Unlock()
	logger.Warn(fmt.Sprintf("try exit session mutex sessionId:%d", sessionId))
	delete(l.sessionMap, sessionId)
	logger.Warn(fmt.Sprintf("exit session mutex sessionId:%d", sessionId))
}
