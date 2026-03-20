package config

// overrideConfigPath 临时覆盖配置文件路径，返回恢复函数
// 仅用于测试
func overrideConfigPath(path string) func() {
	configPathOverride = path
	return func() {
		configPathOverride = ""
	}
}
