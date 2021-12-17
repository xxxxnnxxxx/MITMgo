package core

type Settings struct {
	Id               string            // 标记ID
	IP               string            // IP
	Port             uint16            // 端口号
	RemoteOutputAddr string            // 远程地址,保存数据
	Hosts            []string          // 指定要过滤的host
	MaxRunTime       int               // 最大执行执行时间(分钟)
	Headers          map[string]string // 过滤的请求添加头
	IgnoreWords      []string          // 忽略包含有某些关键词的url，不作为结果输出
	IsContainHttps   bool              // 是否包含TLS通讯数据
	MessageAddr      string            // 消息地址
	Ca               string            // 保存PEM证书路径
	PriKey           string            // PriKey路径
}

func NewSettings() *Settings {
	return &Settings{
		Headers:     make(map[string]string),
		IgnoreWords: []string{},
		Hosts:       []string{},
	}
}
