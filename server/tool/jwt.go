package tool

import (
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go"
)

// 载荷，可添加自己需要的一些信息
type MyClaims struct {
	ID      string  `json:"id"`
	Permiss []int32 `json:"permiss"`
	jwt.StandardClaims
}

const TokenExpireDuration = time.Hour * 2

var MySecret = []byte("今天测试龙岛游戏")

// GenToken 生成JWT
func GenToken(tID string, permiss []int32) (string, error) {
	// 创建一个我们自己的声明
	c := MyClaims{
		ID:      tID,
		Permiss: permiss,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(TokenExpireDuration).Unix(), // 过期时间
			Issuer:    "name-s",                                   // 签发人
		},
	}
	// 使用指定的签名方法创建签名对象
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	// 使用指定的secret签名并获得完整的编码后的字符串token
	return token.SignedString(MySecret)
}

// ParseToken 解析JWT
func ParseToken(tokenString string) (*MyClaims, error) {
	// 解析token
	token, err := jwt.ParseWithClaims(tokenString, &MyClaims{}, func(token *jwt.Token) (i interface{}, err error) {
		return MySecret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*MyClaims); ok && token.Valid { // 校验token
		return claims, nil
	}
	return nil, errors.New("invalid token")
}
