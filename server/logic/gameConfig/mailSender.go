package gameConfig

import (
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("mailSender", &MailSenderCfgLoader{})
}

type MailSenderCfgLoader struct {
	temp1 map[int32]*MailSenderCfg
}

var _ configLoaderInterface = (*MailSenderCfgLoader)(nil)

func (s *MailSenderCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/mailSender.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*MailSenderCfg)
	data, ok := rawData["mailSender"]
	if !ok || len(data) == 0 {
		return nil
	}

	for _, row := range data {
		var v MailSenderCfg
		v.Id = ParseInt(row["id"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return fmt.Errorf("[gameConfig] load mailSender error duplicate ID:%d", v.Id)
		}
		v.MailId = ParseInt(row["mailId"])
		v.BornDay = ParseInt(row["bornDay"])
		v.ServerDay = ParseInt(row["serverDay"])
		v.Cron = strings.TrimSpace(row["cron"])
		v.Date = strings.TrimSpace(row["date"])
		v.DateLimitTime = strings.TrimSpace(row["dateLimitTime"])
		v.Unlock = ParseIntArray(row["unlock"])
		v.UnlockStop = ParseIntArray(row["unlockStop"])

		s.temp1[v.Id] = &v
	}
	return nil
}

func (s *MailSenderCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return fmt.Errorf("[gameConfig] load mailSender error invalid ID:%d", id)
		}
		if v.MailId <= 0 || GetMailContentCfg(v.MailId) == nil {
			return fmt.Errorf("[gameConfig] load mailSender error invalid mailId:%d,id:%d", v.MailId, id)
		}
		for _, unlockId := range v.Unlock {
			if unlockId != 0 && GetUnlockCfg(unlockId) == nil {
				return fmt.Errorf("[gameConfig] load mailSender error invalid unlock:%d,id:%d", unlockId, id)
			}
		}
		for _, stopId := range v.UnlockStop {
			if stopId != 0 && GetUnlockCfg(stopId) == nil {
				return fmt.Errorf("[gameConfig] load mailSender error invalid unlockStop:%d,id:%d", stopId, id)
			}
		}
	}
	return nil
}

func (s *MailSenderCfgLoader) apply() {
	mailSenderCfg.Store(s.temp1)
}

var mailSenderCfg atomic.Value // map[int32]*MailSenderCfg

// MailSenderCfg 邮件模板触发配置（mailSender 表）
type MailSenderCfg struct {
	Id            int32   `json:"id"`
	MailId        int32   `json:"mailId"`        // mailContent 表中的模板ID
	BornDay       int32   `json:"bornDay"`       // 注册第几天（1=首日，5=第5天）
	ServerDay     int32   `json:"serverDay"`     // 开服第几天（1=开服首日）
	Cron          string  `json:"cron"`          // 周期性发送时间（cron 表达式）
	Date          string  `json:"date"`          // 具体时间（如 2006/1/2 15:04）
	DateLimitTime string  `json:"dateLimitTime"` // 具体时间（如 2006/1/2 15:04）
	Unlock        []int32 `json:"unlock"`        // 解锁条件（全部满足）
	UnlockStop    []int32 `json:"unlockStop"`    // 结束条件（任一满足则不发送）
}

// GetAllMailSenderCfg 获取所有 mailSender 配置
func GetAllMailSenderCfg() map[int32]*MailSenderCfg {
	v := mailSenderCfg.Load()
	if v == nil {
		return nil
	}
	return v.(map[int32]*MailSenderCfg)
}
