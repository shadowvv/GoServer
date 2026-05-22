package eventService

import (
	"time"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/service/logger"
)

const MaxUseEventNum = 300             // 每次tick最多处理的事件数量
const BufferSize = 2000                // 事件通道缓冲区大小
const TickTime = 50 * time.Millisecond // 每次tick间隔 10毫秒

type EventBus struct {
	eventBus    chan GameEvent
	subscribers map[string][]chan GameEvent
	bufferSize  int32
}

func NewEventBus(bufferSize int32) *EventBus {
	return &EventBus{
		eventBus:    make(chan GameEvent, bufferSize),
		subscribers: make(map[string][]chan GameEvent),
		bufferSize:  bufferSize,
	}
}

func (eb *EventBus) Subscribe(eventType ...string) <-chan GameEvent {
	// 使用有缓冲channel
	ch := make(chan GameEvent, eb.bufferSize)
	for _, et := range eventType {
		eb.subscribers[et] = append(eb.subscribers[et], ch)
	}
	return ch
}

func (eb *EventBus) SubscribeAll() <-chan GameEvent {
	ch := make(chan GameEvent, eb.bufferSize)
	eb.subscribers["*"] = append(eb.subscribers["*"], ch)
	return ch
}

func (eb *EventBus) Publish(event GameEvent) {
	select {
	case eb.eventBus <- event:
	default:
		// todo 打印日志
		logger.ErrorBySprintf("event bus buffer is full")
	}

}

func (eb *EventBus) tick() {
	var UseEventNum int32 = 1
	for UseEventNum <= MaxUseEventNum {
		select {
		case event := <-eb.eventBus:
			if subscribers, ok := eb.subscribers[event.GetEventType()]; ok {
				for _, ch := range subscribers {
					select {
					case ch <- event:
					default:
						// todo 打印日志
						logger.ErrorBySprintf("event channel buffer is full")
					}
				}
			}
			// 发送给全局订阅者
			if allSubscribers, ok := eb.subscribers["*"]; ok {
				for _, ch := range allSubscribers {
					select {
					case ch <- event:
					default:
						// todo 打印日志
						logger.ErrorBySprintf("event channel buffer is full")
					}
				}
			}
			UseEventNum++
		default:
			return
		}
	}
}

func (eb *EventBus) Run() {
	ticker := time.NewTicker(TickTime)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			eb.tick()
		}
	}
}

func (eb *EventBus) Start() {
	go eb.Run()
}

func (eb *EventBus) SubmitHeroLevelUpEvent(objectID int64, heroId int32, oldLevel, newLevel int32) {
	eb.Publish(NewHeroLevelUpEvent(objectID, heroId, oldLevel, newLevel))
}
func (eb *EventBus) SubmitAccessoryLevelUpEvent(objectID int64, accessoryID int32, oldLevel, newLevel int32) {
	eb.Publish(NewAccessoryLevelUpEvent(objectID, accessoryID, oldLevel, newLevel))
}
func (eb *EventBus) SubmitKillMonsterEvent(objectID int64, sceneId int32, monsterIds []int32) {
	eb.Publish(NewKillMonsterEvent(objectID, sceneId, monsterIds))
}
func (eb *EventBus) SubmitPassInstanceEvent(objectID int64, serverId int32, instanceTypeIdlId enum.InstanceId, instanceId int32) {
	eb.Publish(NewPassInstanceEvent(objectID, serverId, instanceTypeIdlId, instanceId))
}
func (eb *EventBus) SubmitItemCollectEvent(objectID int64, itemInfoList []*gameConfig.ItemConfig) {
	eb.Publish(NewItemCollectEvent(objectID, itemInfoList))
}
func (eb *EventBus) SubmitLuckyLotteryEvent(objectID int64, lotteryType string, lotteryNum int32, itemInfoList []*gameConfig.ItemConfig) {
	eb.Publish(NewLuckyLotteryEvent(objectID, lotteryType, lotteryNum, itemInfoList))
}
func (eb *EventBus) SubmitHeroStarUpEvent(objectID int64, heroId int32, starLevel int32) {
	eb.Publish(NewHeroStarUpEvent(objectID, heroId, starLevel))
}
func (eb *EventBus) SubmitPlayerLevelUpEvent(objectID int64, serverId, level int32) {
	eb.Publish(NewPlayerLevelUpEvent(objectID, serverId, level))
}
func (eb *EventBus) SubmitJoinInstanceEvent(objectID int64, serverId int32, instanceTypeId enum.InstanceId, instanceId int32) {
	eb.Publish(NewJoinInstanceEvent(objectID, serverId, instanceTypeId, instanceId))
}
func (eb *EventBus) SubmitQuickClaimMachineRewardEvent(objectID int64, serverId int32) {
	eb.Publish(NewQuickClaimMachineRewardEvent(objectID, serverId))
}
func (eb *EventBus) SubmitBuildLevelUpEvent(objectID int64, serverId int32, buildId enum.ArchitectureType, buildLevel int32) {
	eb.Publish(NewBuildLevelUpEvent(objectID, serverId, buildId, buildLevel))
}
func (eb *EventBus) SubmitLoopBoxLevelUpEvent(objectID int64, serverId int32, oldLevel, newLevel, systemExp int32) {
	eb.Publish(NewLoopBoxLevelUpEvent(objectID, serverId, oldLevel, newLevel, systemExp))
}

func (eb *EventBus) SubmitDispatchKillMonsterEvent(objectID int64, serverId int32) {
	eb.Publish(NewDispatchKillMonsterEvent(objectID, serverId))
}

func (eb *EventBus) SubmitPlayerPowerChangeEvent(objectID int64, serverId int32, power int64) {
	eb.Publish(NewPlayerPowerChangeEvent(objectID, serverId, power))
}
func (eb *EventBus) SubmitEquipmentStrongEvent(objectID int64, equipmentOwnID int64, oldLevel, newLevel int32) {
	eb.Publish(NewEquipmentStrongEvent(objectID, equipmentOwnID, oldLevel, newLevel))
}
func (eb *EventBus) SubmitAllianceJoinEvent(objectID int64) {
	eb.Publish(NewAllianceJoinEvent(objectID))
}
func (eb *EventBus) SubmitPetStarUpEvent(objectID int64, petOwnID int64, oldStar, newStar int32) {
	eb.Publish(NewPetStarUpEvent(objectID, petOwnID, oldStar, newStar))
}
func (eb *EventBus) SubmitEquipmentForgeEvent(objectID int64, forgeCount int32) {
	eb.Publish(NewEquipmentForgeEvent(objectID, forgeCount))
}
func (eb *EventBus) SubmitEquipmentWearEvent(objectID int64) {
	eb.Publish(NewEquipmentWearEvent(objectID))
}
func (eb *EventBus) SubmitArenaScoreChangeEvent(objectID int64, score int32) {
	eb.Publish(NewArenaScoreChangeEvent(objectID, score))
}
func (eb *EventBus) SubmitAdChestOpenEvent(objectID int64, openCount int32) {
	eb.Publish(NewAdChestOpenEvent(objectID, openCount))
}
func (eb *EventBus) SubmitMainTaskChangeEvent(objectID int64) {
	eb.Publish(NewMainTaskChangeEvent(objectID))
}
func (eb *EventBus) SubmitStoneAttrLevelUpEvent(objectID int64) {
	eb.Publish(NewStoneAttrLevelUpEvent(objectID))
}

func (eb *EventBus) SubmitAddHeroAlbumEvent(objectID int64, heroID int32) {
	eb.Publish(NewAddHeroAlbumEvent(objectID, heroID))
}
func (eb *EventBus) SubmitPetLevelUpEvent(objectID int64, petOwnID int64, oldLevel, newLevel int32) {
	eb.Publish(NewPetLevelUpEvent(objectID, petOwnID, oldLevel, newLevel))
}
