package gameController

import (
	"errors"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/furniture"
	"github.com/drop/GoServer/server/logic/lumber"
	"github.com/drop/GoServer/server/logic/model"
)

var furnitureService *furniture.FurnitureService

func InitFurnitureService() {
	furnitureService = furniture.NewFurnitureService(handleFurnitureEffectChange)
}

func handleFurnitureEffectChange(player *model.PlayerModel, architectureType int32) error {
	switch architectureType {
	case int32(enum.ARCHITECTURE_TYPE_LUMBERYARD):
		return lumber.Service.BeforeFurnitureEffectChange(player, architectureType)
	default:
		return errors.New("architecture type not supported")
	}
}
