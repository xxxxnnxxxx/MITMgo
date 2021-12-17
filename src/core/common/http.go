package common

import (
	"context"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type HttpModule struct {
	client *http.Client
	do     func(req *http.Request) (*http.Response, error)

	timeout      int    // 请求超时时间
	baseAuthUser string // base auth user
	baseAuthPwd  string // base auth pwd
}

type HttpResult struct {
	ResHeaders http.Header
	Body       string
	Res        *http.Response
	MD5OfBody  string
}

func NewHttpResult() *HttpResult {
	return &HttpResult{
		ResHeaders: make(map[string][]string),
	}
}

func NewHttpModule() (*HttpModule, error) {
	httpModule := &HttpModule{
		timeout: 20,
	}
	var tr = http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	httpModule.client = &http.Client{
		Transport: tr,
	}
	httpModule.do = httpModule.client.Do

	return httpModule, nil
}

func NewHttpModule2(client *http.Client) (*HttpModule, error) {
	httpModule := &HttpModule{
		timeout: 20,
	}

	if client == nil {
		httpModule.client = &http.Client{}
	} else {
		httpModule.client = client
	}

	httpModule.do = httpModule.client.Do
	return httpModule, nil
}

func (p *HttpModule) Release() {
	if p.client != nil {
		p.client.CloseIdleConnections()
	}
}

func (p *HttpModule) Set(timeout int, baseAuthUser string, baseAuthPwd string) {
	p.baseAuthUser = baseAuthUser
	p.baseAuthPwd = baseAuthPwd
	p.timeout = timeout
}

func (p *HttpModule) doRequest(method string, url string, headers map[string]string, postData string) (*HttpResult, error) {
	if p.client.CheckRedirect == nil {
		p.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	req, err := http.NewRequest(strings.ToUpper(method), url, nil)
	if err != nil {
		return nil, err
	}

	// timeout
	ctx, cancel := context.WithTimeout(req.Context(), time.Duration(p.timeout)*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	// headers
	if headers != nil && len(headers) > 0 {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}

	// baseAuth
	if len(p.baseAuthUser) > 0 {
		req.SetBasicAuth(p.baseAuthUser, p.baseAuthPwd)
	}

	// postData
	if len(postData) >= 0 {
		//req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.ContentLength = int64(len(postData))
		req.Body = ioutil.NopCloser(strings.NewReader(postData))
	}

	res, err := p.do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	bodys, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	result := NewHttpResult()
	result.Res = res
	result.ResHeaders = res.Header.Clone()
	// 可能有编码问题
	result.Body = string(bodys)

	return result, nil
}

func (p *HttpModule) GET(url string, headers map[string]interface{}) (*HttpResult, error) {
	tmp := make(map[string]string)
	for k, v := range headers {
		val, ok := v.(string)
		if ok {
			tmp[k] = val
		}
	}
	return p.doRequest("GET", url, tmp, "")
}

func (p *HttpModule) POST(url string, headers map[string]interface{}, postData string) (*HttpResult, error) {
	tmp := make(map[string]string)
	for k, v := range headers {
		val, ok := v.(string)
		if ok {
			tmp[k] = val
		}
	}
	return p.doRequest("POST", url, tmp, postData)
}

func (p *HttpModule) HEAD(url string, headers map[string]interface{}) (*HttpResult, error) {
	tmp := make(map[string]string)
	for k, v := range headers {
		val, ok := v.(string)
		if ok {
			tmp[k] = val
		}
	}
	return p.doRequest("HEAD", url, tmp, "")
}
