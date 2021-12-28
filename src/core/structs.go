package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"mitmgo/src/core/common"
	"net/http"
	"net/url"
	"strings"
)

//easyjson:json
type RequestResult struct {
	Id       string            `json:"id"`
	Method   string            `json:"method"`
	Link     string            `json:"link"`
	Headers  map[string]string `json:"headers"`
	PostData string            `json:"postData"`
	Tag      int               `json:"tag"`  // 标记链接是否
	Hash     string            `json:"hash"` // 结构集的唯一标记
}

type RemoteOutputCrawlResult struct {
	Id     string          `json:"id"`
	Result []RequestResult `json:"result"`
}

func NewRemoteOutputCrawlResult() *RemoteOutputCrawlResult {
	return &RemoteOutputCrawlResult{
		Result: make([]RequestResult, 0),
	}
}

func ToRequestResult(Id string, req *http.Request) (*RequestResult, error) {
	if req == nil {
		return nil, errors.New("request is empty")
	}

	if req.Method == "GET" || req.Method == "POST" {
		crawlResult := &RequestResult{
			Id:      Id,
			Method:  req.Method,
			Link:    req.URL.String(),
			Headers: make(map[string]string),
			Tag:     1,
		}

		for k, _ := range req.Header {
			crawlResult.Headers[k] = req.Header.Get(k)
		}

		if req.Method == "POST" {
			// 数据长度不能超过1M
			if req.ContentLength > 0 && req.ContentLength <= 1024*1024 {
				body, err := ioutil.ReadAll(req.Body)
				if err != nil {
					return crawlResult, nil
				}

				req.Body.Close()
				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))

				if v, ok := crawlResult.Headers["Content-Type"]; ok {
					mediaType, _, err := mime.ParseMediaType(v)
					if err != nil {
						crawlResult.PostData = string(body)
						goto exit
					}
					if mediaType == "application/x-www-form-urlencoded" ||
						mediaType == "text/plain" {
						crawlResult.PostData = string(body)
					} else if strings.HasPrefix(mediaType, "multipart/") {
						crawlResult.PostData = string(body)
					} else {
						crawlResult.PostData = string(body)
						goto exit
					}
				}
			}
		}

	exit:
		return crawlResult, nil
	}

	return nil, nil
}

func (p *RequestResult) IslikeVuejsorAngularLnk() bool {

	if len(p.Link) == 0 {
		return false
	}
	// 判断是否存在/#/标记
	uri, err := url.Parse(p.Link)
	if err != nil {
		return false
	}
	concatUrl := uri.Scheme + "://" + uri.Host + uri.Path + "#" + uri.Fragment

	if strings.ToLower(concatUrl) == strings.ToLower(uri.String()) && strings.Contains(uri.String(), "/#/") {
		return true
	}

	return false
}

func (p *RequestResult) GetstandardFlagUriEx(ignorecase bool) string {
	var result = ""
	if p.IslikeVuejsorAngularLnk() {
		return p.Link
	} else {
		result = p.GetUrlWithoutFragmentEx(ignorecase)
	}

	return result
}

func (p *RequestResult) GetUrlWithoutFragmentEx(ignorecase bool) string {
	if len(p.Link) == 0 {
		return ""
	}
	// 判断是否存在/#/标记
	uri, err := url.Parse(p.Link)
	if err != nil {
		return ""
	}

	var u_url string
	if uri.RawQuery == "" {
		var path = uri.Path
		if len(path) == 0 {
			path = "/"
		}
		u_url = uri.Scheme + "://" + uri.Host + path
	} else {
		// 分析原始查询
		var arguments string
		querys := strings.Split(uri.RawQuery, "&")
		for _, query := range querys {
			args := strings.Split(query, "=")
			if len(args) > 0 {
				arguments += common.TrimEx(args[0])
			}
		}
		u_url = strings.ToLower(uri.Scheme) + "://" + strings.ToLower(uri.Host) + uri.Path + "?" + arguments

	}

	// PostData中的参数
	if len(p.PostData) > 0 {
		postData := ""
		// 获取Content-Type
		content_type := ""
		if v, ok := p.Headers["Content-Type"]; ok {
			mediaType, params, err := mime.ParseMediaType(v)
			if err != nil {
				postData = p.PostData
				goto join
			}

			if mediaType == "application/x-www-form-urlencoded" {
				content_type = "application/x-www-form-urlencoded"
			} else if mediaType == "application/hal+json" ||
				mediaType == "application/json" {
				content_type = "json"
			} else if strings.HasPrefix(mediaType, "multipart/") {
				content_type = "multipart/form-data"

				mr := multipart.NewReader(strings.NewReader(p.PostData), params["boundary"])
				for {
					ptmp, err1 := mr.NextRawPart()
					if err1 == io.EOF {
						break
					}

					if vv, ok := ptmp.Header["Content-Disposition"]; ok {
						_, params2, _ := mime.ParseMediaType(strings.Join(vv, ";"))
						if vv1, ok := params2["name"]; ok {
							postData += vv1
						}
					}
				}
			} else if mediaType == "text/xml" ||
				mediaType == "application/xml" ||
				mediaType == "application/xhtml+xml" ||
				mediaType == "application/atom+xml" {
				content_type = "xml"
			}
		}

	join:
		switch content_type {
		case "xml":
			u_url += common.CalcXMLFeatureStr(p.PostData)
		case "json":
			m := map[string]interface{}{}
			//Parsing/Unmarshalling JSON encoding/json
			err := json.Unmarshal([]byte(p.PostData), &m)
			if err == nil {
				u_url += common.CalcJSONFeatureStr(m)
			}
		case "application/x-www-form-urlencoded":
			postArguments := strings.Split(p.PostData, "&")
			for _, argkv := range postArguments {
				args := strings.Split(argkv, "=")
				if len(args) > 0 {
					u_url += common.TrimEx(args[0])
				}
			}
		case "multipart/form-data":
			u_url += postData

		default:
			postArguments := strings.Split(p.PostData, "&")
			if len(postArguments) == 0 {
				u_url += p.PostData
			} else {
				for _, argkv := range postArguments {
					args := strings.Split(argkv, "=")
					if len(args) > 0 {
						u_url += common.TrimEx(args[0])
					}
				}
			}
		}

	}

	if ignorecase {
		return strings.ToUpper(p.Method) + strings.ToLower(u_url)
	} else {
		return strings.ToUpper(p.Method) + u_url
	}
}

type CrawlerOverDTO struct {
	Id      string `json:"id"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewCrawlerOverDTO(
	id string,
	code int,
	message string,
) *CrawlerOverDTO {
	return &CrawlerOverDTO{
		Id:      id,
		Code:    code,
		Message: message,
	}
}
