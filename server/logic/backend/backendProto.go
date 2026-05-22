package backend

import (
	"github.com/drop/GoServer/server/logic/mail"
)

// GmResp 通用响应
type GmResp struct {
	Code int32       `json:"code"` //返回结果标志为(0  正确  else  错误)
	Msg  string      `json:"msg"`  //结果错误信息(正确时为空字符串"")
	Data interface{} `json:"data"` // 具体响应信息
}

// GmLoginReq 登录
type GmLoginReq struct {
	UserName string `json:"username"`
	PassWord string `json:"pwd"`
}

type GmLoginData struct {
	Token   string  `json:"token"`
	Permiss []int32 `json:"permiss"` //玩家权限
}

// GmUserReq 请求管理用户列表
type GmUserReq struct {
	Uid   int32  `json:"uid"`
	Token string `json:"token"`
}

type GmUserData struct {
	User    string  `json:"user"`
	Pwd     int64   `json:"pwd"`
	Permiss []int32 `json:"permiss"`
}

// GmEditGmUserReq 修改用户权限
type GmEditGmUserReq struct {
	Token   string  `json:"token"`
	Name    string  `json:"name"`
	Uid     string  `json:"uid"`
	Pwd     string  `json:"pwd"`
	Add     int32   `json:"add"` //0 修改权限 1 添加用户 2 修改密码 3 删除用户
	Permiss []int32 `json:"permiss"`
}

type GmEditGmUserData struct {
}

// GmUserInfoReq 查询玩家简单信息
type GmUserInfoReq struct {
	Type    int32  `json:"type"`    //查询账号类型 0 账号 1 uid 2 昵称
	Account string `json:"account"` //查询账号信息
	Token   string `json:"token"`
}

type GmUserInfoData struct {
	UserId              int64  `json:"user_id"`
	Account             string `json:"account"`
	NickName            string `json:"nick_name"`
	RegistrationTime    int64  `json:"registration_time"`
	RegistrationChannel int32  `json:"registration_channel"`
	ServerId            int32  `json:"server_id"`
	RechargeNum         int32  `json:"recharge_num"`
	LastLoginTime       int64  `json:"last_login_time"`
	LastOfflineTime     int64  `json:"last_offline_time"`
	Level               int32  `json:"level"`
	MainLevel           int32  `json:"main_level"`
	FiveVsFLevel        int32  `json:"five_vs_f_level"`
	Power               int64  `json:"power"`
	BanStatus           int32  `json:"ban_status"`  // 0 未封禁 1 封禁
	BanReason           int32  `json:"ban_reason"`  // 封禁理由 （直接由策划和前端对接即可，库里存数字）
	MuteStatus          int32  `json:"mute_status"` // 0 未禁言 1 禁言
	MuteReason          int32  `json:"mute_reason"` // 禁言理由
}

// GmGetFormationReq 查询玩家阵容信息
type GmGetFormationReq struct {
	Uid   int64  `json:"user_id"`
	Token string `json:"token"`
}

type HeroInfo struct {
	HeroId        int32          `json:"hero_id"`
	Class         int32          `json:"class"`
	Level         int32          `json:"level"`
	StarLevel     int32          `json:"star_level"`
	EquipId       []int32        `json:"equip_id"`
	EquipQuality  []int32        `json:"equip_quality"`
	EquipLevel    []int32        `json:"equip_level"`
	AccessoryInfo *AccessoryInfo `json:"accessory_info"`
	Power         int64          `json:"power"`
}
type GmGetFormationData struct {
	HeroInfoList []*HeroInfo `json:"hero_info_list"`
}

type GmGetAccessoryReq struct {
	UserId int64  `json:"user_id"`
	Token  string `json:"token"`
}

type AccessoryInfo struct {
	AccessoryId      int32 `json:"accessory_id"`
	AccessoryQuality int32 `json:"accessory_quality"`
	AccessoryLevel   int32 `json:"accessory_level"`
}

type GmGetAccessoryData struct {
	AccessoryInfo []*AccessoryInfo `json:"accessory_info"`
}

// GmUserMailReq 查询玩家邮件
type GmUserMailReq struct {
	Uid   int64    `json:"uid"`
	Token string   `json:"token"`
	Msl   []string `json:"msl"` //邮件状态列表
	Sts   int64    `json:"sts"` //发送时间 区间 开始
	Ste   int64    `json:"ste"` //发送时间 区间 结束
	Ets   int64    `json:"ets"` //到期时间 区间 开始
	Ete   int64    `json:"ete"` //到期时间 区间 结束
}

type GmUserMailData struct {
	Mid          int64                      `json:"mid"`          // 邮件ID
	Mtp          int32                      `json:"mtp"`          // 邮件类型 邮件类型（1普通 2广告 3官方 4命令 5玩家）
	CfgId        int32                      `json:"cfgid"`        // 邮件配置ID
	Content      string                     `json:"content"`      // 邮件内容
	Title        string                     `json:"title"`        // 邮件标题
	Ms           int32                      `json:"ms"`           // 邮件状态
	Ct           int64                      `json:"ct"`           // 创建时间
	Ot           int64                      `json:"ot"`           // 结束时间(结束时间为0表示永不过期) 毫秒时间戳
	ExpireDays   int32                      `json:"expireDays"`   // 过期时间（发邮件时使用，0表示永不过期） 天数
	IsConvenient bool                       `json:"isConvenient"` // 是否可一键领取（true=可一键领取；false=只能单独领取）
	IsExpired    int32                      `json:"is_expired"`   // 是否过期 0 未过期 1 过期
	Del          int32                      `json:"del"`          // 是否删除 0 未删除 1 删除
	Items        []*mail.MailAttachmentItem `json:"items"`        // 附件物品条目列表（业务约定：只有一个附件，直接存 items）
	Sn           string                     `json:"sn"`           // 发送者名称
	Sa           string                     `json:"sa"`           // 发送人头像
}

// pb.MailAttachmentItem 格式与下列一致
//type GmMailAttachmentItem struct {
//	Type int32 `json:"type"` // （回包会有数据，但是发的时候，这里不用写东西，没用，之后优化会删掉这个字段）
//	ID   int32 `json:"id"`   // 道具/货币/资源ID
//	Num  int32 `json:"num"`  // 数量
//}

// GmSendMailReq 发送邮件给玩家
type GmSendMailReq struct {
	Ul          []int64         `json:"ul"` //玩家ID列表 （列表为空表示全服邮件）
	Token       string          `json:"token"`
	Info        *GmUserMailData `json:"info"`        // 发送邮件时 mailId、邮件状态、创建时间、是否过期、是否删除 不用写
	SendType    int32           `json:"sendType"`    // 发送类型：0=自定义玩家 1=全服 2=区服 3=联盟
	ServerIds   []int32         `json:"serverIds"`   // 区服ID列表（SendType=2时使用）
	AllianceIds []int64         `json:"allianceIds"` // 联盟ID列表（SendType=3时使用）
}

type GmSendMailData struct {
}

// GmServerListReq 请求服务器列表
type GmServerListReq struct {
	Uid   int32  `json:"uid"`
	Token string `json:"token"`
}

type GmServerListData struct {
	ServerId         int32  `json:"s"`          //服务器ID
	ServerName       string `json:"n"`          // 服务器名称
	ServerNameId     int32  `json:"sn"`         //服务器名称id
	ServerOpenTime   int64  `json:"st"`         //开始时间
	ServerTime       int64  `json:"serverTime"` //服务器时间，特殊功能使用
	ServerLogicId    int32  `json:"serverLogicId"`
	AreaId           int32  `json:"area"` //大区ID
	AreaName         string `json:"an"`   //大区名称
	MaxRegisterCount int32  `json:"mp"`   //最大玩家数
	RegisterCount    int32  `json:"np"`   //当前玩家数
	TodayActPlayer   int32  `json:"dau"`  //当日活跃用户
	OpenToNewWeight  int32  `json:"openToNewWeight"`
	OpenToNew        int32  `json:"openToNew"` // 是否开放注册
	CanSeeGroupId    int32  `json:"canSeeGroupId"`
	Status           int32  `json:"status"` // 服务器状态（0:正常,1:维护)
}

type GmEditClientVersionReq struct {
	Token             string               `json:"token"`
	ClientVersionList *GmClientVersionData `json:"versionInfo"`
}
type GmEditClientVersionData struct {
}

type GmClientVersionData struct {
	Version      string `json:"v"`             // 客户端版本
	HotFixConfig string `json:"hotfix_config"` // 热更配置
	Examine      int32  `json:"examine"`       // 是否审核
}

type GmGetClientVersionReq struct {
	Token             string                 `json:"token"`
	ClientVersionList []*GmClientVersionData `json:"versionInfo"`
}

type GmGetClientVersionData struct{}

// GmEditServerReq 修改服务器信息
type GmEditServerReq struct {
	Uid   int32             `json:"uid"`
	Token string            `json:"token"`
	Info  *GmServerListData `json:"info"`
}

// GmUserItemChgReq 请求用户资产流水列表
type GmUserItemChgReq struct {
	Uid   int64  `json:"uid"`
	Token string `json:"token"`
	It    int32  `json:"it"` //物品类型
	Id    int32  `json:"id"` //物品ID
	Ft    int32  `json:"ft"` // 来源类型
	Fv    int32  `json:"fv"` //来源值
	St    int64  `json:"st"` //开始时间
	Et    int64  `json:"et"` //结束时间
}

type GmUserItemData struct {
	Uid  int64  `json:"uid"`
	T    int64  `json:"t"`    //添加时间
	It   int32  `json:"it"`   //物品类型
	Id   int32  `json:"id"`   //物品ID
	Ft   int32  `json:"ft"`   // 来源类型
	Fv   int32  `json:"fv"`   //来源值
	Ext  int64  `json:"ext"`  //额外值
	Init string `json:"init"` //初始值
	Chg  string `json:"chg"`  //变化值
	Fina string `json:"fina"` //最终值
}

type GmGamePublicData struct {
	Id           int32  `json:"id"`          //公告ID
	AnnounceType int32  `json:"type"`        //类型
	ShowType     int32  `json:"show_type"`   // 显示页签类型
	Title        string `json:"title"`       // 公告标题
	Content      string `json:"content"`     // 公告内容
	PicAddress   string `json:"pic_address"` // 图片地址
	ServerIds    string `json:"server_Id"`   // 服务器id （ ‘|’ 分割）
	Unlocks      string `json:"unlocks"`     // （ ‘|’ 分割）
	UnlockStop   string `json:"unlock_stop"` // （ ‘|’ 分割）
	StartTime    int64  `json:"start_time"`  // 开始时间
	EndTime      int64  `json:"end_time"`    // 结束时间
	Valid        int32  `json:"valid"`       // 是否生效
	ExtraInfo    string `json:"extra_info"`  // 额外信息
}

// GmUserOrderReq 查询玩家订单
type GmUserOrderReq struct {
	T     int32   `json:"t"`
	Uid   int64   `json:"uid"`
	Token string  `json:"token"`
	Msl   []int32 `json:"msl"`
	Cts   int64   `json:"cts"` //创建时间 区间 开始
	Cte   int64   `json:"cte"` //创建时间 区间 结束
	Pts   int64   `json:"pts"` //支付时间 区间 开始
	Pte   int64   `json:"pte"` //支付时间 区间 结束
}

type GmUserOrderData struct {
	Oid   int64  `json:"oid"`   //订单ID
	Uid   int64  `json:"uid"`   //玩家ID
	Goods int32  `json:"goods"` //商品ID
	P     int32  `json:"p"`     //价格
	Ct    int64  `json:"ct"`    //创建时间
	S     int32  `json:"s"`     //订单状态
	Ot    int64  `json:"ot"`    //支付时间
	Pt    int32  `json:"pt"`    //支付方式
	Bz    string `json:"bz"`    //备注
}

// GmGetRankListReq 获取排行榜列表（支持多条件过滤）
type GmGetRankListReq struct {
	Uid      int32  `json:"uid"`
	Token    string `json:"token"`
	RankId   int32  `json:"rank_id"`   // 排行榜 ID（可选，0 表示不过滤）
	ActId    int32  `json:"act_id"`    // 活动 ID（可选，0 表示不过滤）
	ServerId int32  `json:"server_id"` // 服务器 ID（可选，0 表示不过滤）
	Date     string `json:"date"`      // 日期 YYYYMMDD（可选，空字符串表示不过滤）
}

// GmGetRankListData 排行榜列表条目
type GmGetRankListData struct {
	TableName string `json:"table_name"` // 表名（可直接传给 getRank 接口查询）
	RankId    int32  `json:"rank_id"`    // 排行榜 ID
	ActId     int32  `json:"act_id"`     // 活动 ID（0 为长期榜）
	ServerId  int32  `json:"server_id"`  // 服务器 ID
	Date      string `json:"date"`       // 日期（如果有）
}

// GmGetRankReq 获取排行榜数据（前端传表名，后端直接查该表）
type GmGetRankReq struct {
	Uid       int32  `json:"uid"`
	Token     string `json:"token"`
	TableName string `json:"table_name"` // 排行榜表名（从 getRankList 返回的 table_name）
}

// GmGetRankData 排行榜单条数据
type GmGetRankData struct {
	Rank         int32  `json:"rank"`           // 排名
	UserId       int64  `json:"user_id"`        // 用户ID
	NickName     string `json:"nick_name"`      // 昵称
	Score        int64  `json:"score"`          // 分数
	ThumbUpCount int32  `json:"thumb_up_count"` // 点赞数
	EnterTime    int64  `json:"enter_time"`     // 进入时间
}

// GmGamePublicReq 请求游戏内加载界面公告
type GmGamePublicReq struct {
	Uid   int64  `json:"uid"`
	Token string `json:"token"`
}

// GmEditGamePublicReq 编辑游戏内界面公告
type GmEditGamePublicReq struct {
	Uid   int64             `json:"uid"`
	Token string            `json:"token"`
	Data  *GmGamePublicData `json:"data"`
}

type GmEditGamePublicData struct{}

// GmGetTalkReq 获取聊天列表
type GmGetTalkReq struct {
	ServerId int32  `json:"server_id"` // 不能为空
	Token    string `json:"token"`
	KeyWords string `json:"key_words"`
}

// GmGetUserInventoryReq 获取玩家背包请求
type GmGetUserInventoryReq struct {
	Uid   int64  `json:"user_id"`
	Token string `json:"token"`
}

// GmInventoryItem 背包物品数据
type GmInventoryData struct {
	ItemId  int32 `json:"item_id"`
	ItemNum int64 `json:"item_num"`
}

// GmGetServerActivityConfigReq 获取服务器活动配置请求
type GmGetServerActivityConfigReq struct {
	Token string `json:"token"`
}

// GmServerActivityConfigData 服务器活动配置数据
type GmServerActivityConfigData struct {
	Id             int32  `json:"id"`
	ServerType     int32  `json:"server_type"`
	ServerUnit     string `json:"server_unit"`
	UnlockId       string `json:"unlock_id"`
	AttendUnlockId string `json:"attend_unlock_id"`
	EventOpen      string `json:"event_open"`
	EventEnd       string `json:"event_end"`
	WeekOpen       string `json:"week_open"`
	MonthOpen      string `json:"month_open"`
	Duration       string `json:"duration"`
	SettleTime     int32  `json:"settle_time"`
	IfFirst        int32  `json:"if_first"`
	NextId         int32  `json:"next_id"`
	Cd             int32  `json:"cd"`
	OpenLoopNum    int32  `json:"open_loop_num"`
	IfBlockServer  string `json:"if_block_server"`
	IfBlock        int32  `json:"if_block"`
}

// GmEditServerActivityConfigReq 编辑服务器活动配置请求
type GmEditServerActivityConfigReq struct {
	Token string                      `json:"token"`
	Data  *GmServerActivityConfigData `json:"data"`
}

// GmEditBanUserReq 封号/解封请求
type GmEditBanUserReq struct {
	Token   string         `json:"token"`
	BanInfo *GmBanUserData `json:"ban_info"`
}

// GmBanListData 封号列表数据
type GmBanUserData struct {
	Account   string `json:"account"`
	ServerId  int32  `json:"server_id"`
	Reason    int32  `json:"reason"`
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
}

// GmEditMuteUserReq 禁言/解禁请求
type GmEditUserChatReq struct {
	Token        string              `json:"token"`
	UserChatData *GmEditUserChatData `json:"ban_info"`
}

// GmMuteUserData 禁言数据
type GmEditUserChatData struct {
	Account   string `json:"account"`
	ServerId  int32  `json:"server_id"`
	Reason    int32  `json:"reason"`
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
}

// GmGetUserLogListReq 查询用户操作日志请求
type GmGetUserLogListReq struct {
	Token         string `json:"token"`
	UserId        int64  `json:"uid"`            // 用户ID
	OperationType int32  `json:"operation_type"` // 操作类型（可选，0表示不过滤）
	St            int64  `json:"st"`             // 开始时间（毫秒时间戳，可选）
	Et            int64  `json:"et"`             // 结束时间（毫秒时间戳，可选）
}

// GmUserLogData 用户操作日志数据
type GmUserLogData struct {
	UserId        int64 `json:"user_id"`
	AddTime       int64 `json:"add_time"`
	OperationType int32 `json:"operation_type"`
	Param1        int32 `json:"param1"`
	Param2        int32 `json:"param2"`
	Param3        int32 `json:"param3"`
	Param4        int32 `json:"param4"`
}

// GmExportPlayerReq 导出玩家数据请求
type GmExportPlayerReq struct {
	UserId int64  `json:"user_id"` // 玩家UID
	Token  string `json:"token"`
}

// GmExportPlayerData 导出玩家数据响应（结构化JSON）
type GmExportPlayerData struct {
	Sql  string `json:"sql"`  // 兼容旧格式（保留）
	Json string `json:"json"` // 结构化JSON导出数据
}

// GmImportPlayerReq 导入玩家数据请求
type GmImportPlayerReq struct {
	Token          string `json:"token"`
	Sql            string `json:"sql"`              // 兼容旧格式
	Json           string `json:"json"`             // 结构化JSON导入数据
	TargetAccount  string `json:"target_account"`   // 目标玩家账号
	TargetServerId int32  `json:"target_server_id"` // 目标玩家服务器ID
	CreateTime     int64  `json:"create_time"`      // 创号时间（毫秒），0表示使用文件中的数据
}

// GmImportPlayerData 导入玩家数据响应
type GmImportPlayerData struct {
	UserId    int64  `json:"user_id"`     // 新生成的 user_id
	OldUserId int64  `json:"old_user_id"` // 原来的 user_id
	Msg       string `json:"msg"`
}

// GmKickPlayerReq 踢人请求
type GmKickPlayerReq struct {
	Token string `json:"token"`
	Type  int32  `json:"type"`  // 踢人类型 1=按userid踢人 2=区服踢人 3=全服踢人
	Param int32  `json:"param"` // 参数（type=1时为userid，type=2时为serverId，type=3时无意义）
}

// GmGetThroughputReq 获取吞吐量监控数据请求
type GmGetThroughputReq struct {
	Token string `json:"token"`
}

// GmThroughputItem 单个吞吐量数据
type GmThroughputItem struct {
	Key      string `json:"key"`      // Redis key
	Handled  int64  `json:"handled"`  // 已处理数量
	Received int64  `json:"received"` // 已接收数量
}
