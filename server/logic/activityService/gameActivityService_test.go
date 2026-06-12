package activityService

import (
	"math"
	"testing"

	"github.com/go-redis/redis/v8"

	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/service/dbService"
	"github.com/drop/GoServer/server/tool"
)

func TestGameActivityServiceQueriesRespectActivityBlock(t *testing.T) {
	const (
		serverId   int32 = 1
		activityId int32 = 1001
	)

	tests := []struct {
		name          string
		ifBlock       int32
		ifBlockServer string
		wantOpen      bool
	}{
		{name: "activity is not blocked", wantOpen: true},
		{name: "activity is globally blocked", ifBlock: 1, wantOpen: false},
		{name: "activity is blocked for server", ifBlockServer: "1|2", wantOpen: false},
		{name: "activity is blocked for another server", ifBlockServer: "2|3", wantOpen: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configModel := model.NewServerActivityConfigModel([]*model.ServerActivityConfigEntity{
				{
					Id:            activityId,
					IfBlock:       tt.ifBlock,
					IfBlockServer: tt.ifBlockServer,
				},
			})
			openActivityModel := model.NewServerOpenActivityModel(map[int32][]*model.ServerOpenActivityEntity{
				serverId: {
					{
						ActivityId:   activityId,
						OpenServerId: serverId,
						OpenTime:     tool.UnixNowMilli() - tool.HOUR_MILLI,
						SettleTime:   math.MaxInt64,
						EndTime:      math.MaxInt64,
					},
				},
			})
			service := &GameActivityService{
				serverActivityConfigModel: configModel,
				serverOpenActivityModel:   openActivityModel,
			}

			allOpen := service.GetAllOpenActivityByServerId(serverId)
			if got := len(allOpen) == 1; got != tt.wantOpen {
				t.Fatalf("GetAllOpenActivityByServerId() open = %t, want %t", got, tt.wantOpen)
			}
			if got := service.IsActivityOpen(serverId, activityId) != nil; got != tt.wantOpen {
				t.Fatalf("IsActivityOpen() open = %t, want %t", got, tt.wantOpen)
			}
		})
	}
}

func TestGameActivityServiceReloadPreservesSnapshotsOnRedisFailure(t *testing.T) {
	const (
		serverId   int32 = 1
		activityId int32 = 1001
	)
	service := &GameActivityService{
		serverActivityConfigModel: model.NewServerActivityConfigModel([]*model.ServerActivityConfigEntity{
			{Id: activityId},
		}),
		serverOpenActivityModel: model.NewServerOpenActivityModel(map[int32][]*model.ServerOpenActivityEntity{
			serverId: {
				{
					ActivityId:   activityId,
					OpenServerId: serverId,
					OpenTime:     tool.UnixNowMilli() - tool.HOUR_MILLI,
					SettleTime:   math.MaxInt64,
					EndTime:      math.MaxInt64,
				},
			},
		}),
	}

	previousRDB := dbService.RDB
	failedRDB := redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})
	if err := failedRDB.Close(); err != nil {
		t.Fatalf("close redis client: %v", err)
	}
	dbService.RDB = failedRDB
	t.Cleanup(func() {
		dbService.RDB = previousRDB
	})

	service.Reload()

	if service.GetActivityConfig(activityId) == nil {
		t.Fatal("Reload() removed the existing activity config after Redis failure")
	}
	if service.IsActivityOpen(serverId, activityId) == nil {
		t.Fatal("Reload() removed the existing open activity after Redis failure")
	}
}
