package gameConfig

import (
	"fmt"
	"sync/atomic"

	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("mailContent", &MailContentCfgLoader{})
}

type MailContentCfgLoader struct {
	temp1 map[int32]*MailContentCfg
}

var _ configLoaderInterface = (*MailContentCfgLoader)(nil)

func (s *MailContentCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/mailContent.json`, &rawData); err != nil {
		return err
	}
	s.temp1 = make(map[int32]*MailContentCfg)
	data, ok := rawData["mailContent"]
	if !ok || len(data) == 0 {
		return nil
	}
	for _, row := range data {
		var v MailContentCfg
		v.ID = ParseInt(row["id"])
		if v.ID <= 0 {
			continue
		}
		v.MailType = ParseInt(row["mailType"])
		v.MailTitle = ParseInt(row["mailTitle"])
		v.MailWords = ParseInt(row["mailWords"])
		v.MailExpTime = ParseInt(row["mailExpTime"])
		v.SendName = ParseInt(row["sendName"])
		v.Item = ParseItemArray(row["item"])
		v.IsConvenient = ParseInt(row["isConvenient"]) == 1
		v.IsShow = ParseInt(row["isShow"]) == 1
		if s.temp1[v.ID] != nil {
			return fmt.Errorf("[gameConfig] load mailContent error duplicate ID:%d", v.ID)
		}
		s.temp1[v.ID] = &v
	}
	return nil
}

func (s *MailContentCfgLoader) checkData() error {
	return nil
}

func (s *MailContentCfgLoader) apply() {
	mailContent.Store(s.temp1)
}

var mailContent atomic.Value

// MailContentCfg 邮件模板配置（与表 mailContent 对应）
type MailContentCfg struct {
	ID           int32         // 邮件ID（模板ID）
	MailType     int32         // 邮件类型
	MailTitle    int32         // 邮件标题文本ID
	MailWords    int32         // 邮件内容文本ID
	MailExpTime  int32         // 有效时间（单位：小时，0 表示永不过期）
	SendName     int32         // 寄件人名字文本ID
	Item         []*ItemConfig // 道具附件（格式：道具ID~数量，多个用 | 分隔）
	IsConvenient bool          // 是否便捷领取
	IsShow       bool          // 是否隐藏
}

// GetMailContentCfg 根据邮件模板ID获取配置
func GetMailContentCfg(id int32) *MailContentCfg {
	v := mailContent.Load()
	if v == nil {
		return nil
	}
	return v.(map[int32]*MailContentCfg)[id]
}
