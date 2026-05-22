package gameConfig

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("avatar", &AvatarCfgLoader{})
}

type AvatarCfgLoader struct {
	temp map[int32]*AvatarItemCfg
}

var _ configLoaderInterface = (*AvatarCfgLoader)(nil)

func (s *AvatarCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/avatar.json`, &rawData); err != nil {
		return err
	}

	type subDef struct {
		key      string
		cfgType  enum.AvatarType
		hasTpye  bool
		typeName string
	}
	subs := []subDef{
		{"bubble", enum.AvatarTypeBubble, false, "bubble"},
		{"head", enum.AvatarTypeHead, true, "head"},
		{"headFrame", enum.AvatarTypeHeadFrame, false, "headFrame"},
		{"image", enum.AvatarTypeImage, true, "image"},
		{"title", enum.AvatarTypeTitle, false, "title"},
	}

	s.temp = make(map[int32]*AvatarItemCfg)
	for _, sub := range subs {
		for _, row := range rawData[sub.key] {
			var v AvatarItemCfg
			v.Id = ParseInt(row["id"])
			v.CfgType = sub.cfgType
			v.Name = ParseInt(row["name"])
			v.Quality = ParseInt(row["quality"])
			v.ItemId = ParseInt(row["itemId"])
			v.Attr = ParseIntArray(row["attr"])
			v.AttrNum = ParseIntArray(row["attrNum"])
			if sub.hasTpye {
				v.Type = ParseInt(row["type"])
				v.UnlockId = ParseInt(row["unlockId"])
			}
			if v.Id <= 0 {
				continue
			}
			if s.temp[v.Id] != nil {
				return errors.New(fmt.Sprintf("[gameConfig] load %s error duplicate ID:%d", sub.typeName, v.Id))
			}
			if len(v.Attr) != len(v.AttrNum) {
				return errors.New(fmt.Sprintf("[gameConfig] load %s error attr length not equal attrNum length", sub.typeName))
			}
			s.temp[v.Id] = &v
		}
	}

	return nil
}

func (s *AvatarCfgLoader) checkData() error {
	for id, v := range s.temp {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load avatar error invalid ID:%d", id))
		}
		if v.UnlockId != 0 && GetUnlockCfg(v.UnlockId) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load avatar error invalid UnlockId:%d", v.UnlockId))
		}
	}
	return nil
}

func (s *AvatarCfgLoader) apply() {
	avatarItems.Store(s.temp)
}

var avatarItems atomic.Value

type AvatarItemCfg struct {
	// 外观id
	Id int32 `json:"id"`
	// 子类型(1气泡 2头像 3头像框 4形象 5称号)
	CfgType enum.AvatarType `json:"cfgType"`
	// 类型(仅头像/形象使用)
	Type int32 `json:"type"`
	// 解锁条件(仅头像/形象使用)
	UnlockId int32 `json:"unlockId"`
	// 名称
	Name int32 `json:"name"`
	// 品质
	Quality int32 `json:"quality"`
	// 关联道具id
	ItemId int32 `json:"itemId"`
	// 拥有属性
	Attr []int32 `json:"attr"`
	// 附带属性数值
	AttrNum []int32 `json:"attrNum"`
}

func GetAvatarItemCfg(id int32) *AvatarItemCfg {
	cfgMap := avatarItems.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*AvatarItemCfg)[id]
}

func GetAllAvatarItemCfg() map[int32]*AvatarItemCfg {
	cfgMap := avatarItems.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*AvatarItemCfg)
}
