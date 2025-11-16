package enum

type Environment string

const (
	ENV_DEVELOP Environment = "dev"
	ENV_TEST    Environment = "test"
	ENV_STAGE   Environment = "stage"
	ENV_PRODUCT Environment = "prod"
)

type ServerType string

const (
	SERVER_TYPE_GATE ServerType = "gate"
	SERVER_TYPE_GAME ServerType = "game"
)

type DBPoolType string

const (
	DB_POOL_TYPE_LOGIN DBPoolType = "login"
	DB_POOL_TYPE_SCENE DBPoolType = "scene"
)
