package goproxy

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/google/martian/v3"
	"github.com/google/martian/v3/auth"
	mlog "github.com/google/martian/v3/log"
	"github.com/google/martian/v3/mitm"
	"io/ioutil"
	"log"
	"math"
	"mitmgo/src/core"
	"mitmgo/src/core/common"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

var mediaType = map[string]struct{}{
	"gif":   struct{}{},
	"png":   struct{}{},
	"jpeg":  struct{}{},
	"mp4":   struct{}{},
	"mp3":   struct{}{},
	"avi":   struct{}{},
	"webp":  struct{}{},
	"wav":   struct{}{},
	"webm":  struct{}{},
	"bmp":   struct{}{},
	"ico":   struct{}{},
	"jpg":   struct{}{},
	"jfif":  struct{}{},
	"css":   struct{}{},
	"js":    struct{}{},
	"svg":   struct{}{},
	"woff":  struct{}{},
	"woff2": struct{}{},
	"exe":   struct{}{},
	"zip":   struct{}{},
	"xlsx":  struct{}{},
	"xls":   struct{}{},
	"docx":  struct{}{},
	"doc":   struct{}{},
	"7z":    struct{}{},
	"rtf":   struct{}{},
	"vbs":   struct{}{},
	"rar":   struct{}{},
}

type ProxyEntity struct {
	Id                  string // id
	IP                  string // ip
	Port                uint16 //  端口
	Headers             map[string]string
	Hosts               []string
	IgnoreWords         []string
	IsContainHttps      bool          // 是否包含https请求
	RemoteOutputAddr    string        // 远程输出地址
	Timeout             time.Duration // 连接超时时间
	KeepAlive           time.Duration // 保持连接时间
	TLSHandShakeTimeout time.Duration // 握手超时时间
	MaxRunTime          int           // 最大运行时间
	BeginRunTime        time.Time     // 开始运行时间
	IsExpireTip         bool          // 是否到期提醒
	ResultSet           *common.Stack // 保存结果，只保存生成的JSON字符串
	proxy               *martian.Proxy
	ca                  string
	prikey              string
	resultHash          map[string]struct{} // 保存结果hash
	lock_resultHash     sync.Mutex
}

func NewProxyEntity(Id string,
	ip string,
	port uint16,
	headers map[string]string,
	hosts []string,
	ignoreWords []string,
	isContainHttps bool,
	remoteOutputAddr string,
	timeout time.Duration,
	keepAlive time.Duration,
	tlsHandShakeTimeout time.Duration,
	maxruntime int,
	ca string,
	prikey string) *ProxyEntity {

	if len(ip) == 0 {
		ip = "0.0.0.0"
	}

	if port <= 0 || port >= 65535 {
		port = 8080
	}

	if timeout <= 0 {
		timeout = 30
	}
	if keepAlive <= 0 {
		keepAlive = 30
	}
	if tlsHandShakeTimeout <= 0 {
		tlsHandShakeTimeout = 10
	}

	mlog.SetLevel(0)

	mitmproxy := ProxyEntity{
		Id:                  Id,
		IP:                  ip,
		Port:                port,
		Timeout:             timeout * time.Second,
		KeepAlive:           keepAlive * time.Second,
		TLSHandShakeTimeout: tlsHandShakeTimeout * time.Second,
		Headers:             make(map[string]string),
		IgnoreWords:         []string{},
		IsContainHttps:      isContainHttps,
		RemoteOutputAddr:    remoteOutputAddr,
		MaxRunTime:          maxruntime,
		ResultSet:           common.NewStack(),
		ca:                  ca,
		prikey:              prikey,
		proxy:               martian.NewProxy(),
		resultHash:          make(map[string]struct{}),
	}

	for k, v := range headers {
		mitmproxy.Headers[k] = v
	}

	// hosts
	mitmproxy.Hosts = append(mitmproxy.Hosts, hosts...)

	mitmproxy.IgnoreWords = append(mitmproxy.IgnoreWords, ignoreWords...)

	return &mitmproxy
}

func (p *ProxyEntity) StartServer() error {
	l, err := net.Listen("tcp", p.IP+":"+strconv.Itoa((int)(p.Port)))
	if err != nil {
		return err
	}

	p.BeginRunTime = time.Now()
	tr := &http.Transport{
		Dial: (&net.Dialer{
			Timeout:   p.Timeout,
			KeepAlive: p.KeepAlive,
		}).Dial,
		TLSHandshakeTimeout:   p.TLSHandShakeTimeout,
		ExpectContinueTimeout: 20 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
	}

	p.proxy.SetRoundTripper(tr)

	if p.IsContainHttps {
		var x509c *x509.Certificate
		var priv interface{}

		tlsc, err := tls.LoadX509KeyPair(p.ca, p.prikey)
		if err != nil {
			log.Fatal(err)
		}
		priv = tlsc.PrivateKey

		x509c, err = x509.ParseCertificate(tlsc.Certificate[0])
		if err != nil {
			log.Fatal(err)
		}

		if x509c != nil && priv != nil {
			mc, err := mitm.NewConfig(x509c, priv)
			if err != nil {
				log.Fatal(err)
			}

			p.proxy.SetMITM(mc)
		}
	}
	// add modifier
	//stack, _ := httpspec.NewStack("martian")
	//
	//topg := fifo.NewGroup()
	//pa := proxyauth.NewModifier()
	//pa.SetRequestModifier(p)
	//pa.SetResponseModifier(p)
	//stack.AddRequestModifier(pa)
	//stack.AddResponseModifier(pa)
	//topg.AddResponseModifier(stack)
	//topg.AddRequestModifier(stack)
	//
	//p.proxy.SetRequestModifier(topg)
	//p.proxy.SetResponseModifier(topg)
	p.proxy.SetRequestModifier(p)
	p.proxy.SetResponseModifier(p)

	log.Printf("starting proxy on %s ", l.Addr().String())

	go p.proxy.Serve(l)

	return nil
}

func (p *ProxyEntity) Close() {
	p.proxy.Close()
}

func (p *ProxyEntity) ModifyRequest(req *http.Request) error {

	ctx := martian.NewContext(req)
	actx := auth.FromContext(ctx)

	actx.SetID(id(req.Header))

	if req.Method == "GET" || req.Method == "POST" {
		if len(p.Hosts) > 0 {
			var bFind = false
			for _, k := range p.Hosts {
				if strings.Trim(strings.ToLower(req.URL.Host), " ") == strings.Trim(strings.ToLower(k), " ") {
					bFind = true
				}
			}

			// 没有找到匹配
			if !bFind {
				return nil
			}
		}
		if len(p.IgnoreWords) > 0 {
			bFind := false
			for _, word := range p.IgnoreWords {
				bFind = strings.Contains(req.URL.String(), word)
				if bFind {
					return nil
				}
			}
		}

		func() error {
			crawlResult, err := core.ToCrawlResult(p.Id, req)
			if err != nil {
				return nil
			}
			if crawlResult == nil {
				return nil
			}

			// 去重
			fret := func() error {
				p.lock_resultHash.Lock()
				defer p.lock_resultHash.Unlock()
				unid, _ := common.ToMD5Str(crawlResult.GetstandardFlagUriEx(true))
				if _, ok := p.resultHash[unid]; !ok {
					p.resultHash[unid] = struct{}{}
				} else {
					return errors.New("error")
				}

				crawlResult.Hash = unid

				return nil
			}()

			if fret != nil {
				return nil
			}

			if len(p.RemoteOutputAddr) > 0 {
				remoteResult := core.NewRemoteOutputCrawlResult()
				remoteResult.Id = p.Id
				remoteResult.Result = append(remoteResult.Result, *crawlResult)
				resultStr := common.ToJsonEncodeStruct(remoteResult)

				// send data to the remote addr
				httpModule, err := common.NewHttpModule()
				if err != nil {
					log.Println(err)
					return nil
				}

				httpRes, err := httpModule.POST(p.RemoteOutputAddr,
					map[string]interface{}{
						"Content-Type": "application/json",
					}, resultStr)

				if err != nil {
					fmt.Println("Post the results to the server: " + err.Error())
				}

				if httpRes != nil && len(httpRes.Body) > 0 {
					fmt.Println("Return message: " + httpRes.Body)
				}

				httpModule.Release()
				// 保存结果到结果集中
				p.ResultSet.Push(resultStr)
			} else {
				printResultStr := common.ToJsonEncodeStruct(crawlResult)
				fmt.Print(printResultStr + "\r\n")
				// 保存结果到结果集中
				p.ResultSet.Push(printResultStr)
			}

			return nil

		}()
	}

	return nil
}

func (p *ProxyEntity) ModifyResponse(res *http.Response) error {
	ctx := martian.NewContext(res.Request)
	actx := auth.FromContext(ctx)
	if got, want := actx.ID(), "admin:admin"; got != want {
		//res.StatusCode = http.StatusProxyAuthRequired
		//res.Header.Set("Proxy-Authenticate", "Basic")
		//res.Body =
		//
		//res.ContentLength = 0
		//UpdateResponse(res)

		//return errors.New("sddd")
	}

	if p.MaxRunTime >= 1 {
		runtime := time.Now().Sub(p.BeginRunTime)
		expireTime := int(math.Floor(runtime.Minutes()))

		// 结束前的一分钟提醒
		if expireTime >= p.MaxRunTime-1 {
			p.IsExpireTip = true
			newcontent := `
	<html>
	<head>
	<meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
	</head>
	<body>
		<h1>测试服务马上过期</h1>
	</body>
	</html>
`
			//body, err := ioutil.ReadAll(res.Body)
			//
			//if err != nil {
			//	return err
			//}
			res.StatusCode = 200
			if res.ContentLength > 0 {
				err := res.Body.Close()
				if err != nil {
					return err
				}
			}

			res.Body = ioutil.NopCloser(bytes.NewReader([]byte(newcontent)))
			res.ContentLength = int64(len(newcontent))
			res.Header.Set("Content-Encoding", "identity")
			res.Header.Set("Content-Length", strconv.Itoa(len(newcontent)))
			res.Header.Set("Content-Type", "text/html;charset=UTF-8")

		}
	}

	return nil
}

func UpdateResponse(r *http.Response) error {
	b, _ := ioutil.ReadAll(r.Body)
	buf := bytes.NewBufferString("Monkey")
	buf.Write(b)
	r.Body = ioutil.NopCloser(buf)
	r.StatusCode = http.StatusProxyAuthRequired
	r.Header["Proxy-Authenticate"] = []string{"Basic"}
	r.Header["Content-Length"] = []string{fmt.Sprint(buf.Len())}
	return nil
}

func id(header http.Header) string {
	id := strings.TrimPrefix(header.Get("Proxy-Authorization"), "Basic ")

	data, err := base64.StdEncoding.DecodeString(id)
	if err != nil {
		return ""
	}

	return string(data)
}
