package activityService

import (
	"math"
	"testing"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/tool"
)

func TestFindServerUnitIndex(t *testing.T) {
	serverUnits := []*model.ServerUnitVector{
		{Left: 1, Right: 10},
		{Units: []int32{20, 21, 22}},
		{Left: 30, Right: 40},
	}

	tests := []struct {
		name      string
		serverId  int32
		wantIndex int32
		wantFind  bool
	}{
		{name: "first range", serverId: 5, wantIndex: 0, wantFind: true},
		{name: "explicit server in second group", serverId: 21, wantIndex: 1, wantFind: true},
		{name: "third range boundary", serverId: 40, wantIndex: 2, wantFind: true},
		{name: "not in any group", serverId: 25, wantFind: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			index, find := findServerUnitIndex(serverUnits, tt.serverId)
			if index != tt.wantIndex || find != tt.wantFind {
				t.Fatalf("findServerUnitIndex() = (%d, %t), want (%d, %t)", index, find, tt.wantIndex, tt.wantFind)
			}
		})
	}
}

func TestRunGatewayActivityHeartbeatRecoversPanic(t *testing.T) {
	callCount := 0

	runGatewayActivityHeartbeat(func() {
		callCount++
		panic("test panic")
	})
	runGatewayActivityHeartbeat(func() {
		callCount++
	})

	if callCount != 2 {
		t.Fatalf("heartbeat call count = %d, want 2", callCount)
	}
}

func TestShouldReloadGameActivity(t *testing.T) {
	tests := []struct {
		name                string
		configChanged       bool
		activityChangeCount int
		want                bool
	}{
		{name: "no changes", want: false},
		{name: "config changed without activity changes", configChanged: true, want: true},
		{name: "activity changed", activityChangeCount: 1, want: true},
		{name: "config and activity changed", configChanged: true, activityChangeCount: 1, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldReloadGameActivity(tt.configChanged, tt.activityChangeCount); got != tt.want {
				t.Fatalf("shouldReloadGameActivity() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestCheckConfigChangeSettleTime(t *testing.T) {
	const (
		openTime    int64 = 1_750_000_000_000
		currentTime int64 = openTime + tool.HOUR_MILLI
	)
	service := &GatewayActivityService{}

	tests := []struct {
		name           string
		duration       int32
		settleTime     int32
		initialSettle  int64
		expectedSettle int64
	}{
		{
			name:           "infinite duration with settle time",
			duration:       0,
			settleTime:     4,
			initialSettle:  math.MaxInt64,
			expectedSettle: openTime + 4*tool.HOUR_MILLI,
		},
		{
			name:           "finite duration without settle time",
			duration:       8,
			settleTime:     0,
			initialSettle:  openTime + 4*tool.HOUR_MILLI,
			expectedSettle: math.MaxInt64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &model.ServerActivityConfigEntity{
				SettleTime:    tt.settleTime,
				DurationTimes: []int32{tt.duration},
			}
			activity := &model.ServerOpenActivityEntity{
				OpenTime:   openTime,
				SettleTime: tt.initialSettle,
				EndTime:    math.MaxInt64,
			}

			changed := service.checkConfigChange(config, activity, currentTime)
			if !changed {
				t.Fatal("checkConfigChange() changed = false, want true")
			}
			if activity.SettleTime != tt.expectedSettle {
				t.Fatalf("checkConfigChange() settle time = %d, want %d", activity.SettleTime, tt.expectedSettle)
			}
		})
	}
}

func TestGetActivityDuration(t *testing.T) {
	tests := []struct {
		name          string
		durationTimes []int32
		openCount     int32
		want          int32
	}{
		{name: "empty duration", openCount: 1, want: 0},
		{name: "first opening", durationTimes: []int32{2, 4, 6}, openCount: 1, want: 2},
		{name: "second opening", durationTimes: []int32{2, 4, 6}, openCount: 2, want: 4},
		{name: "opening exceeds configured durations", durationTimes: []int32{2, 4, 6}, openCount: 5, want: 6},
		{name: "unknown opening count uses first duration", durationTimes: []int32{2, 4, 6}, openCount: 0, want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getActivityDuration(tt.durationTimes, tt.openCount); got != tt.want {
				t.Fatalf("getActivityDuration() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCalculateActivityTimes(t *testing.T) {
	const openTime int64 = 1_750_000_000_000

	tests := []struct {
		name       string
		config     *model.ServerActivityConfigEntity
		openCount  int32
		wantSettle int64
		wantEnd    int64
	}{
		{
			name: "first opening",
			config: &model.ServerActivityConfigEntity{
				SettleTime:    3,
				DurationTimes: []int32{2, 4},
			},
			openCount:  1,
			wantSettle: openTime + 3*tool.HOUR_MILLI,
			wantEnd:    openTime + 2*tool.HOUR_MILLI,
		},
		{
			name: "duration uses current opening",
			config: &model.ServerActivityConfigEntity{
				SettleTime:    3,
				DurationTimes: []int32{2, 4},
			},
			openCount:  2,
			wantSettle: openTime + 3*tool.HOUR_MILLI,
			wantEnd:    openTime + 4*tool.HOUR_MILLI,
		},
		{
			name: "duration falls back to last value",
			config: &model.ServerActivityConfigEntity{
				SettleTime:    3,
				DurationTimes: []int32{2, 4},
			},
			openCount:  5,
			wantSettle: openTime + 3*tool.HOUR_MILLI,
			wantEnd:    openTime + 4*tool.HOUR_MILLI,
		},
		{
			name: "activity never settles or ends",
			config: &model.ServerActivityConfigEntity{
				SettleTime:    0,
				DurationTimes: []int32{0},
			},
			openCount:  1,
			wantSettle: math.MaxInt64,
			wantEnd:    math.MaxInt64,
		},
		{
			name: "empty duration never ends",
			config: &model.ServerActivityConfigEntity{
				SettleTime: 3,
			},
			openCount:  1,
			wantSettle: openTime + 3*tool.HOUR_MILLI,
			wantEnd:    math.MaxInt64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settleTime, endTime := calculateActivityTimes(tt.config, openTime, tt.openCount)
			if settleTime != tt.wantSettle || endTime != tt.wantEnd {
				t.Fatalf("calculateActivityTimes() = (%d, %d), want (%d, %d)", settleTime, endTime, tt.wantSettle, tt.wantEnd)
			}
		})
	}
}

func TestCheckActivityOpenCalculatesTimesFromConfig(t *testing.T) {
	const currentTime int64 = 1_750_000_000_000
	service := &GatewayActivityService{}
	config := &model.ServerActivityConfigEntity{
		Id:             1001,
		ServerType:     int32(enum.ActivityServerType_Single),
		ServerUnitInfo: &model.ServerUnitData{AllServer: true},
		SettleTime:     3,
		DurationTimes:  []int32{2, 4},
	}

	openInfo, opened := service.checkActivityOpen(config, &model.GameServerInfoEntity{ServerId: 1}, 0, currentTime, false)
	if !opened {
		t.Fatal("checkActivityOpen() opened = false, want true")
	}
	if openInfo.OpenTime != currentTime {
		t.Fatalf("checkActivityOpen() open time = %d, want %d", openInfo.OpenTime, currentTime)
	}
	if openInfo.SettleTime != currentTime+3*tool.HOUR_MILLI {
		t.Fatalf("checkActivityOpen() settle time = %d, want %d", openInfo.SettleTime, currentTime+3*tool.HOUR_MILLI)
	}
	if openInfo.EndTime != currentTime+2*tool.HOUR_MILLI {
		t.Fatalf("checkActivityOpen() end time = %d, want %d", openInfo.EndTime, currentTime+2*tool.HOUR_MILLI)
	}
}

func TestCheckConfigChangeUsesCurrentOpeningDuration(t *testing.T) {
	const (
		openTime    int64 = 1_750_000_000_000
		currentTime int64 = openTime + tool.HOUR_MILLI
	)
	service := &GatewayActivityService{}

	tests := []struct {
		name      string
		openCount int32
		wantHours int64
	}{
		{name: "uses current opening duration", openCount: 2, wantHours: 4},
		{name: "uses last duration when opening count exceeds config", openCount: 5, wantHours: 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &model.ServerActivityConfigEntity{
				DurationTimes: []int32{2, 4, 6},
			}
			activity := &model.ServerOpenActivityEntity{
				OpenTime:   openTime,
				SettleTime: math.MaxInt64,
				EndTime:    openTime + 10*tool.HOUR_MILLI,
				OpenCount:  tt.openCount,
			}

			changed := service.checkConfigChange(config, activity, currentTime)
			if !changed {
				t.Fatal("checkConfigChange() changed = false, want true")
			}
			wantEndTime := openTime + tt.wantHours*tool.HOUR_MILLI
			if activity.EndTime != wantEndTime {
				t.Fatalf("checkConfigChange() end time = %d, want %d", activity.EndTime, wantEndTime)
			}
		})
	}
}

func TestCheckConfigChangeDoesNotUpdateFinishedTimes(t *testing.T) {
	const (
		openTime    int64 = 1_750_000_000_000
		currentTime int64 = openTime + 10*tool.HOUR_MILLI
	)
	service := &GatewayActivityService{}
	config := &model.ServerActivityConfigEntity{
		SettleTime:    20,
		DurationTimes: []int32{20},
	}
	activity := &model.ServerOpenActivityEntity{
		OpenTime:   openTime,
		SettleTime: currentTime - tool.HOUR_MILLI,
		EndTime:    currentTime - tool.HOUR_MILLI,
		OpenCount:  1,
	}

	changed := service.checkConfigChange(config, activity, currentTime)
	if changed {
		t.Fatal("checkConfigChange() changed = true, want false")
	}
	if activity.SettleTime != currentTime-tool.HOUR_MILLI || activity.EndTime != currentTime-tool.HOUR_MILLI {
		t.Fatalf("checkConfigChange() updated finished times: %+v", activity)
	}
}

func TestCheckActivityOpenMatchesAnyServerUnit(t *testing.T) {
	const currentTime int64 = 1_750_000_000_000
	serverUnits := []*model.ServerUnitVector{
		{Left: 1, Right: 10},
		{Units: []int32{20, 21, 22}},
	}
	service := &GatewayActivityService{}

	tests := []struct {
		name        string
		serverType  enum.ActivityServerType
		serverId    int32
		wantOpen    bool
		versionUnit int32
	}{
		{name: "single server matches second group", serverType: enum.ActivityServerType_Single, serverId: 21, wantOpen: true, versionUnit: 21},
		{name: "multi server matches second group", serverType: enum.ActivityServerType_Multi, serverId: 21, wantOpen: true, versionUnit: 1},
		{name: "server does not match any group", serverType: enum.ActivityServerType_Single, serverId: 15, wantOpen: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &model.ServerActivityConfigEntity{
				Id:         1001,
				ServerType: int32(tt.serverType),
				ServerUnitInfo: &model.ServerUnitData{
					ServerUnitVector: serverUnits,
				},
			}
			openInfo, opened := service.checkActivityOpen(config, &model.GameServerInfoEntity{ServerId: tt.serverId}, 0, currentTime, false)
			if opened != tt.wantOpen {
				t.Fatalf("checkActivityOpen() opened = %t, want %t", opened, tt.wantOpen)
			}
			if !tt.wantOpen {
				return
			}
			wantVersion := getActivityVersion(tool.GetTodayDataStringByTimeStamp(currentTime), tt.versionUnit, 1)
			if openInfo.Version != wantVersion {
				t.Fatalf("checkActivityOpen() version = %q, want %q", openInfo.Version, wantVersion)
			}
		})
	}
}

func TestCheckActivityOpenCount(t *testing.T) {
	const (
		currentTime int64 = 1_750_000_000_000
		serverId    int32 = 1
	)
	service := &GatewayActivityService{}
	server := &model.GameServerInfoEntity{ServerId: serverId}

	tests := []struct {
		name          string
		openLoopNum   int32
		openCount     int32
		wantOpen      bool
		wantOpenCount int32
	}{
		{name: "non-loop activity opens once", openCount: 0, wantOpen: true, wantOpenCount: 1},
		{name: "non-loop activity does not reopen", openCount: 1, wantOpen: false},
		{name: "loop activity opens next round", openLoopNum: 3, openCount: 1, wantOpen: true, wantOpenCount: 2},
		{name: "loop activity stops at limit", openLoopNum: 3, openCount: 3, wantOpen: false},
		{name: "infinite loop activity ignores open count", openLoopNum: -1, openCount: 100, wantOpen: true, wantOpenCount: 101},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &model.ServerActivityConfigEntity{
				Id:          1001,
				ServerType:  int32(enum.ActivityServerType_Single),
				OpenLoopNum: tt.openLoopNum,
				ServerUnitInfo: &model.ServerUnitData{
					AllServer: true,
				},
			}

			openInfo, opened := service.checkActivityOpen(config, server, tt.openCount, currentTime, false)
			if opened != tt.wantOpen {
				t.Fatalf("checkActivityOpen() opened = %t, want %t", opened, tt.wantOpen)
			}
			if !tt.wantOpen {
				return
			}
			wantVersion := getActivityVersion(tool.GetTodayDataStringByTimeStamp(currentTime), serverId, tt.wantOpenCount)
			if openInfo.Version != wantVersion {
				t.Fatalf("checkActivityOpen() version = %q, want %q", openInfo.Version, wantVersion)
			}
			if openInfo.OpenCount != tt.wantOpenCount {
				t.Fatalf("checkActivityOpen() open count = %d, want %d", openInfo.OpenCount, tt.wantOpenCount)
			}
		})
	}
}

func TestRefreshAllActivityOpensNextActivityInChain(t *testing.T) {
	const (
		serverId       int32 = 1
		firstActivity  int32 = 1001
		secondActivity int32 = 1002
	)
	configModel := model.NewServerActivityConfigModel([]*model.ServerActivityConfigEntity{
		{
			Id:          firstActivity,
			ServerType:  int32(enum.ActivityServerType_Single),
			NextId:      secondActivity,
			IfFirst:     1,
			OpenLoopNum: 3,
		},
		{
			Id:          secondActivity,
			ServerType:  int32(enum.ActivityServerType_Single),
			OpenLoopNum: 1,
		},
	})
	service := &GatewayActivityService{
		serverActivityConfigModel: configModel,
	}
	configs := configModel.GetAllServerActivityConfig()
	servers := map[int32]*model.GameServerInfoEntity{
		serverId: {ServerId: serverId},
	}
	activities := map[int32]map[int32]*model.ServerOpenActivityEntity{
		serverId: {
			firstActivity: {
				ActivityId:   firstActivity,
				OpenServerId: serverId,
				EndTime:      tool.UnixNowMilli() - tool.HOUR_MILLI,
				OpenCount:    1,
			},
		},
	}

	changed := service.refreshAllActivity(configs, servers, activities, false)
	nextActivity := changed[serverId][secondActivity]
	if nextActivity == nil {
		t.Fatalf("refreshAllActivity() did not open next activity, changed = %+v", changed)
	}
	if nextActivity.ActivityId != secondActivity {
		t.Fatalf("next activity id = %d, want %d", nextActivity.ActivityId, secondActivity)
	}
	if _, exists := changed[serverId][firstActivity]; exists {
		t.Fatalf("refreshAllActivity() reopened first activity while advancing the chain")
	}
}

func TestRefreshAllActivitySkipsChainForRemovedServer(t *testing.T) {
	const (
		serverId       int32 = 1
		firstActivity  int32 = 1001
		secondActivity int32 = 1002
	)
	configModel := model.NewServerActivityConfigModel([]*model.ServerActivityConfigEntity{
		{
			Id:         firstActivity,
			ServerType: int32(enum.ActivityServerType_Single),
			NextId:     secondActivity,
		},
		{
			Id:          secondActivity,
			ServerType:  int32(enum.ActivityServerType_Single),
			OpenLoopNum: 1,
		},
	})
	service := &GatewayActivityService{
		serverActivityConfigModel: configModel,
	}
	activities := map[int32]map[int32]*model.ServerOpenActivityEntity{
		serverId: {
			firstActivity: {
				ActivityId:   firstActivity,
				OpenServerId: serverId,
				EndTime:      tool.UnixNowMilli() - tool.HOUR_MILLI,
				OpenCount:    1,
			},
		},
	}

	changed := service.refreshAllActivity(
		configModel.GetAllServerActivityConfig(),
		map[int32]*model.GameServerInfoEntity{},
		activities,
		false,
	)
	if len(changed) != 0 {
		t.Fatalf("refreshAllActivity() changed activities for removed server: %+v", changed)
	}
}

func TestRefreshAllActivityWaitsForChainCooldown(t *testing.T) {
	const (
		serverId       int32 = 1
		firstActivity  int32 = 1001
		secondActivity int32 = 1002
	)
	configModel := model.NewServerActivityConfigModel([]*model.ServerActivityConfigEntity{
		{
			Id:         firstActivity,
			ServerType: int32(enum.ActivityServerType_Single),
			NextId:     secondActivity,
			Cd:         2,
		},
		{
			Id:          secondActivity,
			ServerType:  int32(enum.ActivityServerType_Single),
			OpenLoopNum: 1,
		},
	})
	service := &GatewayActivityService{
		serverActivityConfigModel: configModel,
	}
	configs := configModel.GetAllServerActivityConfig()
	servers := map[int32]*model.GameServerInfoEntity{
		serverId: {ServerId: serverId},
	}
	activities := map[int32]map[int32]*model.ServerOpenActivityEntity{
		serverId: {
			firstActivity: {
				ActivityId:   firstActivity,
				OpenServerId: serverId,
				EndTime:      tool.UnixNowMilli() - tool.HOUR_MILLI,
				OpenCount:    1,
			},
		},
	}

	changed := service.refreshAllActivity(configs, servers, activities, false)
	if len(changed) != 0 {
		t.Fatalf("refreshAllActivity() changed activities before cooldown elapsed: %+v", changed)
	}
}

func TestRefreshAllActivityStopsChainAtOpenLimit(t *testing.T) {
	const (
		serverId       int32 = 1
		firstActivity  int32 = 1001
		secondActivity int32 = 1002
	)
	configModel := model.NewServerActivityConfigModel([]*model.ServerActivityConfigEntity{
		{
			Id:         firstActivity,
			ServerType: int32(enum.ActivityServerType_Single),
			NextId:     secondActivity,
		},
		{
			Id:          secondActivity,
			ServerType:  int32(enum.ActivityServerType_Single),
			OpenLoopNum: 1,
		},
	})
	service := &GatewayActivityService{
		serverActivityConfigModel: configModel,
	}
	configs := configModel.GetAllServerActivityConfig()
	servers := map[int32]*model.GameServerInfoEntity{
		serverId: {ServerId: serverId},
	}
	activities := map[int32]map[int32]*model.ServerOpenActivityEntity{
		serverId: {
			firstActivity: {
				ActivityId:   firstActivity,
				OpenServerId: serverId,
				EndTime:      tool.UnixNowMilli() - tool.HOUR_MILLI,
				OpenCount:    1,
			},
			secondActivity: {
				ActivityId:   secondActivity,
				OpenServerId: serverId,
				EndTime:      tool.UnixNowMilli() - tool.HOUR_MILLI,
				OpenCount:    1,
			},
		},
	}

	changed := service.refreshAllActivity(configs, servers, activities, false)
	if len(changed) != 0 {
		t.Fatalf("refreshAllActivity() advanced chain past open limit: %+v", changed)
	}
}

func TestRefreshAllActivityWaitsWhileSequentialChainIsActive(t *testing.T) {
	const (
		serverId       int32 = 1
		firstActivity  int32 = 1001
		secondActivity int32 = 1002
	)
	configModel := model.NewServerActivityConfigModel([]*model.ServerActivityConfigEntity{
		{
			Id:          firstActivity,
			ServerType:  int32(enum.ActivityServerType_Single),
			NextId:      secondActivity,
			IfFirst:     1,
			OpenLoopNum: 3,
		},
		{
			Id:          secondActivity,
			ServerType:  int32(enum.ActivityServerType_Single),
			OpenLoopNum: 3,
		},
	})
	service := &GatewayActivityService{serverActivityConfigModel: configModel}
	currentTime := tool.UnixNowMilli()
	activities := map[int32]map[int32]*model.ServerOpenActivityEntity{
		serverId: {
			firstActivity: {
				ActivityId: firstActivity,
				EndTime:    currentTime - tool.HOUR_MILLI,
				OpenCount:  1,
			},
			secondActivity: {
				ActivityId: secondActivity,
				EndTime:    currentTime + tool.HOUR_MILLI,
				OpenCount:  1,
			},
		},
	}

	changed := service.refreshAllActivity(
		configModel.GetAllServerActivityConfig(),
		map[int32]*model.GameServerInfoEntity{serverId: {ServerId: serverId}},
		activities,
		false,
	)
	if len(changed) != 0 {
		t.Fatalf("refreshAllActivity() changed activities while sequential chain was active: %+v", changed)
	}
}

func TestRefreshAllActivityReopensFirstAfterSequentialChainFinishes(t *testing.T) {
	const (
		serverId       int32 = 1
		firstActivity  int32 = 1001
		secondActivity int32 = 1002
	)
	configModel := model.NewServerActivityConfigModel([]*model.ServerActivityConfigEntity{
		{
			Id:          firstActivity,
			ServerType:  int32(enum.ActivityServerType_Single),
			NextId:      secondActivity,
			IfFirst:     1,
			OpenLoopNum: 3,
		},
		{
			Id:          secondActivity,
			ServerType:  int32(enum.ActivityServerType_Single),
			OpenLoopNum: 3,
		},
	})
	service := &GatewayActivityService{serverActivityConfigModel: configModel}
	currentTime := tool.UnixNowMilli()
	activities := map[int32]map[int32]*model.ServerOpenActivityEntity{
		serverId: {
			firstActivity: {
				ActivityId: firstActivity,
				EndTime:    currentTime - tool.HOUR_MILLI,
				OpenCount:  1,
			},
			secondActivity: {
				ActivityId: secondActivity,
				EndTime:    currentTime - tool.HOUR_MILLI,
				OpenCount:  1,
			},
		},
	}

	changed := service.refreshAllActivity(
		configModel.GetAllServerActivityConfig(),
		map[int32]*model.GameServerInfoEntity{serverId: {ServerId: serverId}},
		activities,
		false,
	)
	if changed[serverId][firstActivity] == nil {
		t.Fatalf("refreshAllActivity() did not reopen first activity after sequential chain finished: %+v", changed)
	}
	if _, exists := changed[serverId][secondActivity]; exists {
		t.Fatalf("refreshAllActivity() reopened a completed downstream activity: %+v", changed)
	}
}

func TestRefreshAllActivityCircularChainReopensFirstOnlyFromTail(t *testing.T) {
	const (
		serverId       int32 = 1
		firstActivity  int32 = 1001
		secondActivity int32 = 1002
	)
	configModel := model.NewServerActivityConfigModel([]*model.ServerActivityConfigEntity{
		{
			Id:          firstActivity,
			ServerType:  int32(enum.ActivityServerType_Single),
			NextId:      secondActivity,
			IfFirst:     1,
			OpenLoopNum: 3,
		},
		{
			Id:          secondActivity,
			ServerType:  int32(enum.ActivityServerType_Single),
			NextId:      firstActivity,
			OpenLoopNum: 3,
		},
	})
	service := &GatewayActivityService{serverActivityConfigModel: configModel}
	currentTime := tool.UnixNowMilli()
	activities := map[int32]map[int32]*model.ServerOpenActivityEntity{
		serverId: {
			firstActivity: {
				ActivityId: firstActivity,
				EndTime:    currentTime - tool.HOUR_MILLI,
				OpenCount:  1,
			},
			secondActivity: {
				ActivityId: secondActivity,
				EndTime:    currentTime - tool.HOUR_MILLI,
				OpenCount:  1,
			},
		},
	}

	changed := service.refreshAllActivity(
		configModel.GetAllServerActivityConfig(),
		map[int32]*model.GameServerInfoEntity{serverId: {ServerId: serverId}},
		activities,
		false,
	)
	if changed[serverId][firstActivity] == nil {
		t.Fatalf("refreshAllActivity() did not reopen circular chain first activity from tail: %+v", changed)
	}
	if _, exists := changed[serverId][secondActivity]; exists {
		t.Fatalf("refreshAllActivity() advanced circular chain from a completed earlier round: %+v", changed)
	}
}
