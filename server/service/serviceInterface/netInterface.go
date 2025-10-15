package serviceInterface

type NetInterface interface {
	GetName() string
	GetIp() string
	GetMac() string
}
