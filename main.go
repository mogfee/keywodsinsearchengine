package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/mogfee/keywodsinsearchengine/chromerun"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"
)

var (
	inFileName string
	domain     string
	outFile    string
)

func init() {
	flag.StringVar(&inFileName, "in", "./in.txt", "输入关键词\n换行")
	flag.StringVar(&domain, "domain", "echinacities.com", "检测域名")
	flag.StringVar(&outFile, "out", "./out.data", "倒出数据")
}

type KeywordIndex struct {
	Keyword  string    `json:"keyword"`
	Engine   string    `json:"engine"`
	Page     int       `json:"page"`
	Index    int       `json:"index"`
	LastTime time.Time `json:"last_time"`
}

//type Keyword struct {
//	Keyword string       `json:"keyword"`
//	Google  KeywordIndex `json:"google"`
//	BingCN  KeywordIndex `json:"bing_cn"`
//	BingEN  KeywordIndex `json:"bing_en"`
//}

var currentResults = make(map[string]*KeywordIndex)
var keywords []string

//获取关键词
func readInKeywords(inFile string) []string {
	body, err := ioutil.ReadFile(inFile)
	if err != nil {
		panic(err)
	}
	keywors := strings.Split(fmt.Sprintf("%s", body), "\n")
	for _, v := range keywors {
		keywords = append(keywords, strings.TrimSpace(v))
	}
	return keywords
}
func readOutKeywords(inFile string) map[string]*KeywordIndex {

	body, err := ioutil.ReadFile(inFile)
	if err != nil {
		if os.IsNotExist(err) {
			os.Create(inFile)
		} else {
			panic(err)
		}
	}
	result := make(map[string]*KeywordIndex)
	if len(body) == 0 {
		return result
	}
	if err := json.Unmarshal(body, &result); err != nil {
		panic(err)
	}
	return result
}

const (
	GOOGLEENGINE = "google"
	BINGCN       = "bingCN"
	BINGEN       = "bingEN"
)

func getDataKey(searchEngine string, kwd string) string {
	return fmt.Sprintf("%s_%s", searchEngine, kwd)
}
func loadOldData() {
	loadResult := readOutKeywords(outFile)
	min, _ := TimeTodayMinAndMax(time.Now())
	for _, v := range keywords {
		if len(v) == 0 {
			continue
		}

		if vv := loadResult[getDataKey(GOOGLEENGINE, v)]; vv == nil || vv.LastTime.Before(min) {
			currentResults[getDataKey(GOOGLEENGINE, v)] = &KeywordIndex{
				Keyword: v,
				Engine:  GOOGLEENGINE,
			}
		} else {
			currentResults[getDataKey(GOOGLEENGINE, v)] = &KeywordIndex{
				Keyword:  vv.Keyword,
				Engine:   vv.Engine,
				Page:     vv.Page,
				Index:    vv.Index,
				LastTime: vv.LastTime,
			}
		}

		if vv := loadResult[getDataKey(BINGCN, v)]; vv == nil || vv.LastTime.Before(min) {
			currentResults[getDataKey(BINGCN, v)] = &KeywordIndex{
				Keyword: v,
				Engine:  BINGCN,
			}
		} else {
			currentResults[getDataKey(BINGCN, v)] = &KeywordIndex{
				Keyword:  vv.Keyword,
				Engine:   vv.Engine,
				Page:     vv.Page,
				Index:    vv.Index,
				LastTime: vv.LastTime,
			}
		}

		if vv := loadResult[getDataKey(BINGEN, v)]; vv == nil || vv.LastTime.Before(min) {
			currentResults[getDataKey(BINGEN, v)] = &KeywordIndex{
				Keyword: v,
				Engine:  BINGEN,
			}
		} else {
			currentResults[getDataKey(BINGEN, v)] = &KeywordIndex{
				Keyword:  vv.Keyword,
				Engine:   vv.Engine,
				Page:     vv.Page,
				Index:    vv.Index,
				LastTime: vv.LastTime,
			}
		}
	}
}

func loadKeyData(ctx context.Context, searchEngine string, kwd string) (bool, error) {
	res := currentResults[getDataKey(searchEngine, kwd)]
	if res.LastTime.IsZero() {
		resp, err := chromerun.GetResponse(ctx, searchEngine, kwd, domain)
		if err != nil {
			return false, err
		}
		currentResults[getDataKey(searchEngine, kwd)] = &KeywordIndex{
			Page:     resp.Page,
			Index:    resp.Index,
			LastTime: time.Now(),
		}
	}
	return true, nil
}
func loadData(errChan chan error) {
	ctx, _ := chromerun.RunChrome(context.Background(), false)
	go func(errChan chan error) {
		for {
			canBreak := true
			for _, v := range currentResults {
				if v.Keyword == "" {
					continue
				}

				if ok, err := loadKeyData(ctx, GOOGLEENGINE, v.Keyword); err != nil {
					canBreak = false
					fmt.Println(err)
				} else if !ok {
					canBreak = false
				}

				if ok, err := loadKeyData(ctx, BINGCN, v.Keyword); err != nil {
					canBreak = false
					fmt.Println(err)
				} else if !ok {
					canBreak = false
				}

				if ok, err := loadKeyData(ctx, BINGEN, v.Keyword); err != nil {
					canBreak = false
					fmt.Println(err)
				} else if !ok {
					canBreak = false
				}
			}
			if canBreak {
				time.Sleep(time.Second)
			}
		}
		fmt.Println(currentResults)
	}(errChan)
}
func loadNumber(page, num int) string {
	if page == 100 {
		return "<td>50+</td>"
	} else {
		index := (page-1)*10 + num
		if index <= 0 {
			index = 0
		}
		return fmt.Sprintf("<td>%d</td>", index)
	}
}

var cdata = time.Now().Format("20060102")

func loadDay(t time.Time) string {
	if t.Format("20060102") != cdata {
		if t.IsZero() {
			return `<td style="color: red">待加载</td>`

		}
		return `<td style="color: red">` + t.Format("20060102") + `</td>`
	}
	return `<td>` + cdata + `</td>`
}
func runHttp(errChan chan error) {

	srv := http.NewServeMux()
	srv.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		data := ``
		for _, v := range keywords {
			googleInfo := currentResults[getDataKey(GOOGLEENGINE, v)]
			bingCNInfo := currentResults[getDataKey(BINGCN, v)]
			bingENInfo := currentResults[getDataKey(BINGEN, v)]
			data = fmt.Sprintf(`%s 
			<tr><td>%s</td>
			%s %s
			%s %s
			%s %s
			</tr>`, data, v,
				loadNumber(googleInfo.Page, googleInfo.Index), loadDay(googleInfo.LastTime),
				loadNumber(bingCNInfo.Page, bingCNInfo.Index), loadDay(bingCNInfo.LastTime),
				loadNumber(bingENInfo.Page, bingENInfo.Index), loadDay(bingENInfo.LastTime),
			)
		}

		writer.Write([]byte(`<html>
<header>
    <meta charset="utf-8"/>
</header>
<body>
<table border="1">
    <tr>
        <td>关键词</td>
        <td>google</td>
        <td>最后更新时间</td>
        <td>bing 国内</td>
        <td>最后更新时间</td>
        <td>bing 国外</td>
        <td>最后更新时间</td>
    </tr>
    ` + data + `
</table>
</body>
</html>`))
	})

	if err := http.ListenAndServe(":8080", srv); err != nil {
		errChan <- err
	}
}
func main() {
	flag.Parse()

	errChan := make(chan error, 1)
	readInKeywords(inFileName)
	loadOldData()
	//loadData(errChan)
	go runHttp(errChan)
	log.Printf("server listen on http://localhost:8080\n")
	go func() {
		// 等待中断信号以优雅地关闭服务器（设置 5 秒的超时时间）
		signalChan := make(chan os.Signal)
		signal.Notify(signalChan, os.Interrupt)
		errChan <- errors.New(fmt.Sprintf("%v", <-signalChan))
	}()
	<-errChan

	body, _ := json.MarshalIndent(currentResults, " ", " ")
	fmt.Println(ioutil.WriteFile(outFile, body, os.ModePerm))
}

const DefaultLayout = "2006-01-02 15:04:05"

func TimeTodayMinAndMax(ctime time.Time) (min, max time.Time) {
	start, _ := time.ParseInLocation(DefaultLayout, ctime.Format("2006-01-02")+" 00:00:00", time.Local)
	end, _ := time.ParseInLocation(DefaultLayout, ctime.Format("2006-01-02")+" 23:59:59", time.Local)
	return start, end
}
