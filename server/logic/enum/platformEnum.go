package enum

type Environment int32

const (
	ENV_DEVELOP Environment = 0
	ENV_TEST    Environment = 1
	ENV_PRODUCT Environment = 2
)

type ServerType int32

const (
	SERVER_TYPE_GAME  ServerType = 0
	SERVER_TYPE_LOGIC ServerType = 1
)
