package area

import (
	"encoding/json"
	"fmt"
	"github.com/axgle/mahonia"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"time"
)

// 省份正则表达式
// <td><a href='11.html'>北京市<br/></a></td>
const pReg string = "<td><a href='(.*?).html'>(.*?)<br/></a></td>"

// 市级与县级表达式
const casReg string = "<tr class='.*?'><td><a href=.*?>(.*?)</a></td><td><a href=.*?>(.*?)</a></td></tr>"

const host = "http://www.stats.gov.cn/tjsj/tjbz/tjyqhdmhcxhfdm"

var _year string

//Start
//@params year 抓取年份
//@return 已经完成的数据（树形结构）
func Start(year string) []Area {
	_year = year
	province := getProvince()
	for i1, p := range province {
		city := getCity(&p)
		province[i1] = p
		for i2, c := range city {
			county := getCounty(&c)
			city[i2] = c
			for _, v := range county {
				fmt.Printf("%s %s %s \n", p.Name, c.Name, v.Name)
			}
		}
	}
	// 导出json
	WriteJson(province)
	return province
}

// 获取省级地区
// @return areas 地区
func getProvince() []Area {
	// /2019/index.html
	url := fmt.Sprintf("/%s/%s", _year, "index.html")
	areas := fetch(host, url, pReg)
	return areas
}

// 获取市级地区
// @params area 上级地区
// @return 市级地区
// issues: https://github.com/modood/Administrative-divisions-of-China/issues/57
func getCity(area *Area) []Area {
	cCode := area.Code[0:2]
	//url := "/2019/" + cCode + ".html"
	url := fmt.Sprintf("/%s/%s.html", _year, cCode)
	areas := fetch(host, url, casReg)
	area.Areas = areas
	return areas
}

// @Params area 上级地区
// @return areas 地区
// issues: https://github.com/modood/Administrative-divisions-of-China/issues/57
func getCounty(area *Area) []Area {
	cCode := area.Code[0:2]
	aCode := area.Code[0:4]
	//url := "/2019/" + cCode + "/" + aCode + ".html"
	url := fmt.Sprintf("/%s/%s/%s.html", _year, cCode, aCode)
	areas := fetch(host, url, casReg)
	area.Areas = areas
	return areas
}

// 获取网页地区信息
// @params host
// @params route path
// @params reg 表达式
// @params codeLen 编码长度
func fetch(host string, route string, reg string) []Area {
	out := getBody(host, route)
	compile := regexp.MustCompile(reg)
	allString := compile.FindAllStringSubmatch(out, -1)
	areas := make([]Area, len(allString))
	for i, match := range allString {
		areas[i] = Area{match[1], match[2], nil}
	}
	return areas
}

func getBody(host string, route string) string {
	client := &http.Client{}
	for {
		request, err := http.NewRequest("GET", host+route, nil)
		if err != nil {
			fmt.Println("fatal error ", err.Error())
			os.Exit(0)
		}
		request.Header.Add("Accept-Language", "")
		request.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.132 Safari/537.36")
		request.Header.Add("Accept-Charset", "GBK,utf-8;q=0.7,*;q=0.3")
		response, err := client.Do(request)
		if err != nil || response == nil {
			fmt.Println("fatal error")
			panic(err)
		}
		code := response.StatusCode
		// 熔断或者超时或者404等
		if code != 200 {
			fmt.Printf("[Error] %d 休眠 30 秒重试 \n", code)
			time.Sleep(time.Duration(30) * time.Second)
		} else {
			body := response.Body
			return readBody(body)
		}
	}
	return ""
}

// 读取body
func readBody(body io.ReadCloser) string {
	byte2, _ := ioutil.ReadAll(body)
	defer body.Close()
	env := mahonia.NewDecoder("GBK")
	out := env.ConvertString(string(byte2))
	return out
}

// 写入json file
// @params areas 地区
func WriteJson(area []Area) {
	bytes, err := json.Marshal(area)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(0)
	}
	fileName := "dist/area-%d.json"
	currentTime := time.Now().UnixNano() / 1e6
	fileName = fmt.Sprintf(fileName, currentTime)
	err = ioutil.WriteFile(fileName, bytes, os.ModeAppend)
	if err != nil {
		return
	}

}

// 地区
type Area struct {
	Code  string `json:"code"`     //编码
	Name  string `json:"name"`     //名称
	Areas []Area `json:"children"` //下级行政
}
