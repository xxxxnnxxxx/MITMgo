package common

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"github.com/google/uuid"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

func IsFile(path string) bool {
	return !IsDir(path)
}

func IsExist(str string) bool {
	if _, err := os.Stat(str); os.IsNotExist(err) {
		return false
	}

	return true
}

func ReadFileAll(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	content, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func ReadFileBinary(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	content, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return content, nil
}

func ToMD5Str(s string) (string, error) {
	h5 := md5.New()
	_, err := io.WriteString(h5, s)

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(h5.Sum(nil)), nil
}

func ToJsonEncodeStruct(s interface{}) string {
	if s == nil {
		return ""
	}

	byteBuf := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(byteBuf)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(s)
	if err != nil {
		return ""
	}

	return byteBuf.String()
}

func GenerateUniqueStr() string {
	uuidWithHyphen := uuid.New()
	uuid := strings.Replace(uuidWithHyphen.String(), "-", "", -1)

	return uuid
}

func WriteFile(file string, data []byte) (int, error) {
	f, err := os.Create(file)
	if err != nil {
		return -1, err
	}

	writed, err := f.Write(data)
	f.Close()

	return writed, err
}

func GetCurrentDir() (string, error) {
	// 找到该启动当前进程的可执行文件的路径名
	str, err := os.Executable()
	if err != nil {
		return "", err
	}
	str = filepath.Dir(str)

	return str, nil
}

// 移除换行符号
func TrimEx(s string) string {
	if len(s) == 0 {
		return s
	}
	re, err := regexp.Compile(`(^\s+|\s+$)`)
	if err != nil {
		return s
	}

	return re.ReplaceAllString(s, "")
}

func CalcJSONFeatureStr(aMap map[string]interface{}) string {
	result := ""
	for key, val := range aMap {
		switch val.(type) {
		case map[string]interface{}:
			result += key + "{}"
			result += CalcJSONFeatureStr(val.(map[string]interface{}))
		case []interface{}:
			result += key + "[]"
		default:
			result += key
		}
	}

	return result
}

func CalcXMLFeatureStr(content string) string {
	var t xml.Token
	var err error
	result := ""

	inputReader := strings.NewReader(content)

	decoder := xml.NewDecoder(inputReader)
	for t, err = decoder.Token(); err == nil; t, err = decoder.Token() {
		switch token := t.(type) {
		// 处理元素开始（标签）
		case xml.StartElement:
			result += token.Name.Local
		// 处理元素结束（标签）
		case xml.EndElement:
			result += token.Name.Local
		// 处理字符数据（这里就是元素的文本）
		case xml.CharData:

		default:
			// ...
		}
	}

	return result
}
