package xcresult3

import "github.com/bitrise-io/go-utils/log"

func initData(data map[string]interface{}) map[string]interface{} {
	if data == nil {
		data = map[string]interface{}{}
	}
	data["source"] = "deploy-to-bitrise-io"
	return data
}

func logWarn(tag string, data map[string]interface{}, format string, v ...interface{}) {
	log.RWarnf("deploy-to-bitrise-io", tag, initData(data), format, v...)
}
