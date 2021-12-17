package manage

import (
	"encoding/json"
	"errors"
	"fmt"
	opt "github.com/pborman/getopt"
	"log"
	"mitmgo/src/core"
	"mitmgo/src/core/common"
	"mitmgo/src/goproxy"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

var (
	Version   string // 软件版本
	BuildTime string // 创建日期
)

type MITMManager struct {
	Setting *core.Settings
	mitm    *goproxy.ProxyEntity
}

func NewMITMManager() *MITMManager {
	return &MITMManager{
		Setting: core.NewSettings(),
	}
}

func (p *MITMManager) ParseOpt() (bool, error) {
	var isDisplayVersion = false

	opt.StringVarLong(&p.Setting.Id, "id", 'i', `uniquely mark a task. example: --id 62bd64a1-ef71-4db6-a1e2-ca06fa96f97a`)
	opt.StringVarLong(&p.Setting.RemoteOutputAddr, "remote-output-addr", 'R', `the address that receive the result. example: 127.0.0.1:8888`)
	opt.StringVarLong(&p.Setting.IP, "ip", 'I', `proxy server address`)
	opt.Uint16VarLong(&p.Setting.Port, "port", 'P', "special a port for proxy. example: --port 8080")
	hosts := opt.StringLong("hosts", 'T', "", `sepcial a host for filter the request, example: --hosts "[\"admin\", \"admin123\"]"`)
	opt.IntVarLong(&p.Setting.MaxRunTime, "maxruntime", 't', "the time of running the proxy. (unit:hour) defalut:24 hours")
	opt.StringVarLong(&p.Setting.Ca, "cert", 'c', `a path of cert file`)
	opt.StringVarLong(&p.Setting.PriKey, "prikey", 'p', `the prikey of the cert`)
	opt.BoolVarLong(&p.Setting.IsContainHttps, "contain-https", 's', "does it contain the https?")
	headers := opt.StringLong("headers", 'H', "", `the custom http headers for each request, example: --headers "{\"myheader\":\"abc\"}" `)
	ignoreWords := opt.StringLong("ignore-words", 'G', "", `set keywords when  a url which contains will be ignored in result-set. example: --ignore-words "[\"admin\", \"admin123\"]"`)
	generateCA := opt.BoolLong("generate-ca", 'n', `does generate a new ca ?`)
	caDir := opt.StringLong("ca-outputdir", 'o', ``, `output ca and prikey into the directory`)
	opt.BoolVarLong(&isDisplayVersion, "version", 'v', "display the program's version and built-time")
	opt.StringVarLong(&p.Setting.MessageAddr, "message-addr", 'M', "the address which receive the message from this program.example: http://127.0.0.1:4000")
	opt.Parse()

	if isDisplayVersion {
		fmt.Printf("version: %s\r\nbuildtime: %s\r\n", Version, BuildTime)
		return false, nil
	}

	// 生成证书
	if *generateCA {
		var savedir string = "./CA"
		// 判断输出目录是否存在
		if len(*caDir) > 0 {
			// 判断是否存在CA文件夹，如果不存在则创建
			if ok := common.IsExist(*caDir); !ok {
				return false, errors.New("not found the directory")
			}

			if ok := common.IsDir(*caDir); !ok {
				return false, errors.New("it's not a directory")
			}

			savedir = *caDir
		}

		// 判断是否存在CA文件夹，如果不存在则创建
		if ok := common.IsExist(savedir); !ok {
			os.Mkdir("CA", os.ModePerm)
		}

		ca, caprikey, err := core.GenerateCA()
		if err != nil {
			return false, err
		}

		// 保存文件
		capath := filepath.Join(savedir, "ca.pem")
		_, err = common.WriteFile(capath, ca)
		if err != nil {
			return false, err
		}

		caprikeypath := filepath.Join(savedir, "caprikey.pem")
		_, err = common.WriteFile(caprikeypath, caprikey)
		if err != nil {
			return false, err
		}

		return false, nil
	}

	// 如果没有指定证书和私钥的路径，则指定到当前目录下的ca
	if len(p.Setting.Ca) == 0 {
		currentDir, _ := common.GetCurrentDir()
		p.Setting.Ca = filepath.Join(currentDir, "CA", "ca.pem")

	}

	if len(p.Setting.PriKey) == 0 {
		currentDir, _ := common.GetCurrentDir()

		p.Setting.PriKey = filepath.Join(currentDir, "CA", "caprikey.pem")
	}

	// 设置最大运行时间
	if p.Setting.MaxRunTime <= 0 {
		p.Setting.MaxRunTime = 180
	}
	// headers
	if len(*headers) > 0 {
		err := json.Unmarshal([]byte(*headers), &p.Setting.Headers)
		if err != nil {
			return false, err
		}
	}
	// ignore-words
	if len(*ignoreWords) > 0 {
		err := json.Unmarshal([]byte(*ignoreWords), &p.Setting.IgnoreWords)
		if err != nil {
			return false, err
		}
	}
	// hosts
	if len(*hosts) > 0 {
		var hostsArray = make([]string, 0)
		err := json.Unmarshal([]byte(*hosts), &hostsArray)
		if err != nil {
			return false, err
		}

		p.Setting.Hosts = append(p.Setting.Hosts, hostsArray...)
	}

	return true, nil
}

func (p *MITMManager) Initialize() error {
	p.mitm = goproxy.NewProxyEntity(
		p.Setting.Id,
		p.Setting.IP,
		p.Setting.Port,
		p.Setting.Headers,
		p.Setting.Hosts,
		p.Setting.IgnoreWords,
		p.Setting.IsContainHttps,
		p.Setting.RemoteOutputAddr,
		30,
		30,
		10,
		p.Setting.MaxRunTime,
		p.Setting.Ca,
		p.Setting.PriKey,
	)
	return nil
}

func (p *MITMManager) WriteToLog(id string, content string) {
	filename := ""
	if len(id) == 0 {
		filename = common.GenerateUniqueStr()
	} else {
		filename = id
	}

	currentDir, err := common.GetCurrentDir()
	if err != nil {
		return
	}

	logRootDir := filepath.Join(currentDir, "log")
	if !common.IsExist(logRootDir) {
		err := os.Mkdir(logRootDir, os.ModePerm)
		if err != nil {
			return
		}
	}

	logDir := filepath.Join(currentDir, "log", "passivescanner")
	if !common.IsExist(logDir) {
		err := os.Mkdir(logDir, os.ModePerm)
		if err != nil {
			return
		}
	}

	logFile := filepath.Join(logDir, filename)

	common.WriteFile(logFile, []byte(content))
}

func (p *MITMManager) Do() error {
	var ret error = nil
	p.mitm.StartServer()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-time.After(time.Duration(p.Setting.MaxRunTime) * time.Minute):
		if len(p.Setting.MessageAddr) > 0 {
			crawlerOverDTO := core.NewCrawlerOverDTO(p.Setting.Id, 0, "running timeout")
			messageStr := common.ToJsonEncodeStruct(crawlerOverDTO)

			if len(messageStr) > 0 {
				// send data to the remote addr
				httpModule, err := common.NewHttpModule()
				if err != nil {
					log.Println(err)
					ret = err
				}

				httpRes, err := httpModule.POST(p.Setting.MessageAddr,
					map[string]interface{}{
						"Content-Type": "application/json",
					}, messageStr)

				if err != nil {
					fmt.Println(err)
				}

				if httpRes != nil && len(httpRes.Body) > 0 {
					fmt.Println("Return message: " + httpRes.Body)
				}

				httpModule.Release()
			}
		}

		ret = errors.New("running timeout")
	case <-sigc:
		ret = errors.New("user cancel")
	}

	// 保存结果到日志中
	resultSet := ""
	for {
		resultStr := p.mitm.ResultSet.Pop()
		if resultStr == nil {
			break
		}
		if item, ok := resultStr.(string); ok {
			resultSet += "\r\n" + item
		}
	}

	p.WriteToLog(p.Setting.Id, resultSet)

	defer p.mitm.Close()
	return ret
}
