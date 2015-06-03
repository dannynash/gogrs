// 每日收盤後產生符合選股條件的報告.
//
/*
Install:

	go install github.com/toomore/gogrs/cmd/twsereport

Usage:

	twsereport [flags]

The flags are:

	-twse
		上市股票代碼，可使用 ',' 分隔多組代碼，例：2618,2329
	-twsecate
		上市股票類別，可使用 ',' 分隔多組代碼，例：11,15
	-ncpu
		指定 CPU 數量，預設為實際 CPU 數量

*/
package main

import (
	"flag"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/toomore/gogrs/tradingdays"
	"github.com/toomore/gogrs/twse"
)

type checkGroupList []checkGroup

func (c *checkGroupList) Add(f checkGroup) {
	if (*c)[0] == nil {
		(*c)[0] = f
	} else {
		*c = append(*c, f)
	}
}

var (
	wg         sync.WaitGroup
	twseNo     = flag.String("twse", "", "上市股票代碼，可使用 ',' 分隔多組代碼，例：2618,2329")
	twseCate   = flag.String("twsecate", "", "上市股票類別，可使用 ',' 分隔多組代碼，例：11,15")
	ncpu       = flag.Int("ncpu", runtime.NumCPU(), "指定 CPU 數量，預設為實際 CPU 數量")
	ckList     = make(checkGroupList, 1)
	white      = color.New(color.FgWhite, color.Bold).SprintfFunc()
	red        = color.New(color.FgRed, color.Bold).SprintfFunc()
	green      = color.New(color.FgGreen, color.Bold).SprintfFunc()
	yellow     = color.New(color.FgYellow).SprintfFunc()
	yellowBold = color.New(color.FgYellow, color.Bold).SprintfFunc()
	blue       = color.New(color.FgBlue).SprintfFunc()
)

func init() {
	runtime.GOMAXPROCS(*ncpu)
}

func prettyprint(stock *twse.Data, check checkGroup) string {
	var (
		Price       = stock.GetPriceList()[len(stock.GetPriceList())-1]
		RangeValue  = stock.GetRangeList()[len(stock.GetRangeList())-1]
		Volume      = stock.GetVolumeList()[len(stock.GetVolumeList())-1] / 1000
		outputcolor func(string, ...interface{}) string
	)

	switch {
	case RangeValue > 0:
		outputcolor = red
	case RangeValue < 0:
		outputcolor = green
	default:
		outputcolor = white
	}

	return fmt.Sprintf("%s %s %s %s%s %s",
		yellow("[%s]", check),
		blue("%s", stock.RawData[stock.Len()-1][0]),
		outputcolor("%s %s", stock.No, stock.Name),
		outputcolor("$%.2f", Price),
		outputcolor("(%.2f)", RangeValue),
		outputcolor("%d", Volume),
	)
}

func main() {
	flag.Parse()
	var datalist []*twse.Data
	var catelist []twse.StockInfo
	var twselist []string
	var catenolist []string

	if *twseCate != "" {
		l := &twse.Lists{Date: tradingdays.FindRecentlyOpened(time.Now())}

		for _, v := range strings.Split(*twseCate, ",") {
			catelist = l.GetCategoryList(v)
			for _, s := range catelist {
				catenolist = append(catenolist, s.No)
			}
		}
	}

	if *twseNo != "" {
		twselist = strings.Split(*twseNo, ",")
	}
	datalist = make([]*twse.Data, len(twselist)+len(catenolist))

	for i, no := range append(twselist, catenolist...) {
		datalist[i] = twse.NewTWSE(no, tradingdays.FindRecentlyOpened(time.Now()))
	}

	if len(datalist) > 0 {
		for _, check := range ckList {
			fmt.Println(yellowBold("----- %v -----", check))
			wg.Add(len(datalist))
			for _, stock := range datalist {
				go func(check checkGroup, stock *twse.Data) {
					defer wg.Done()
					runtime.Gosched()
					if check.CheckFunc(stock) {
						fmt.Println(prettyprint(stock, check))
					}
				}(check, stock)
			}
			wg.Wait()
		}
	} else {
		flag.PrintDefaults()
	}
}
