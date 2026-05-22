package platform

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"sync"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/platform/nodeConfig"
	"github.com/drop/GoServer/server/service/logger"
)

var pprofOnce sync.Once

func StartPprofByEnv() {
	if nodeConfig.Env != enum.ENV_LOCAL && nodeConfig.Env != enum.ENV_DEVELOP && nodeConfig.Env != enum.ENV_TEST {
		return
	}

	pprofOnce.Do(func() {
		nodeTypeIndex := enum.GetNodeTypeIndex(enum.NodeType(nodeConfig.NodeType))
		port := 16000 + int(nodeTypeIndex)*100 + int(nodeConfig.NodeId)
		addr := fmt.Sprintf(":%d", port)
		go func() {
			logger.InfoWithSprintf("[platform] pprof enabled env:%s addr:%s", nodeConfig.Env, addr)
			if err := http.ListenAndServe(addr, nil); err != nil {
				logger.ErrorBySprintf("[platform] pprof listen error env:%s addr:%s err:%v", nodeConfig.Env, addr, err)
			}
		}()
	})
}
