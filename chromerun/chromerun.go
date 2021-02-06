package chromerun

import (
	"context"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"regexp"
	"strings"
	"time"
)

type Response struct {
	Page    int
	Index   int
	NextUrl string
}

func RunChrome(ctx context.Context, hide bool) (context.Context, context.CancelFunc) {
	options := []chromedp.ExecAllocatorOption{
		chromedp.Flag("hide-scrollbars", false),
		chromedp.Flag("mute-audio", false),
		chromedp.UserAgent(`Mozilla/5.0 (Windows NT 6.3; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.103 Safari/537.36`),
	}
	if !hide {
		options = append(options, chromedp.Flag("headless", false))
	}
	options = append(chromedp.DefaultExecAllocatorOptions[:], options...)
	c, _ := chromedp.NewExecAllocator(ctx, options...)
	return chromedp.NewContext(c)
}

func GetResponse(ctx context.Context, searchEngine string, kwd, domain string) (Response, error) {
	pageIndex := 1
	nextUrl := ""
	resp := Response{}
	var err error
	for {
		if pageIndex > 5 {
			return Response{
				Page:  100,
				Index: 0,
			}, nil
		}
		if searchEngine == "bingCN" {
			resp, err = getBingCNData(kwd, domain, ctx, nextUrl, pageIndex)
		} else if searchEngine == "bingEN" {
			resp, err = getBingENData(kwd, domain, ctx, nextUrl, pageIndex)
		} else if searchEngine == "google" {
			resp, err = getGoogleData(kwd, domain, ctx, nextUrl, pageIndex)
		}
		if err != nil {
			return resp, err
		}
		if resp.Page > 0 {
			return resp, nil
		}
		if resp.NextUrl != "" {
			nextUrl = resp.NextUrl
			pageIndex++
		}
	}
	return resp, nil
}

func getBindResponseData(body, domain string, pageIndex int) (Response, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))

	if err != nil {
		return Response{}, err
	}
	reg, _ := regexp.Compile(domain)
	index := 0
	doc.Find(".b_algo").Each(func(i int, selection *goquery.Selection) {
		arr := reg.FindAllStringSubmatch(selection.Text(), -1)
		if index == 0 && len(arr) > 0 {
			index = i + 1
		}
	})
	if index > 0 {
		return Response{
			Page:  pageIndex,
			Index: index,
		}, nil
	}
	nextUrl, _ := doc.Find(".sb_pagN").Attr("href")
	return Response{
		NextUrl: fmt.Sprintf("https://cn.bing.com%s", nextUrl),
	}, nil
}
func getBingCNData(kwd, domain string, ctx context.Context, url string, pageIndex int) (Response, error) {
	body := ""
	var err error
	if pageIndex == 1 {
		err = chromedp.Run(ctx,
			chromedp.Navigate(`https://cn.bing.com/search`),
			chromedp.WaitVisible(`input[name="q"]`),
			chromedp.SendKeys(`input[name="q"]`, kwd, chromedp.ByQuery),
			chromedp.Click(`#sb_form_go`, chromedp.ByQuery),
			chromedp.OuterHTML(`#b_results`, &body, chromedp.ByID),
		)
	} else {
		err = chromedp.Run(ctx,
			chromedp.Navigate(url),
			chromedp.OuterHTML(`#b_results`, &body, chromedp.ByID),
		)
	}

	if err != nil {
		return Response{}, err
	}
	return getBindResponseData(body, domain, pageIndex)
}
func getBingENData(kwd, domain string, ctx context.Context, url string, pageIndex int) (Response, error) {
	body := ""
	var err error
	if pageIndex == 1 {
		err = chromedp.Run(ctx,
			chromedp.Navigate(`https://cn.bing.com/search`),
			chromedp.Click(`#est_en`, chromedp.ByID),
			chromedp.Sleep(time.Second*1),
			chromedp.WaitVisible(`input[name="q"]`),
			chromedp.SendKeys(`input[name="q"]`, kwd, chromedp.ByQuery),
			chromedp.Click(`#sb_form_go`, chromedp.ByQuery),
			chromedp.OuterHTML(`#b_results`, &body, chromedp.ByID),
		)
	} else {
		err = chromedp.Run(ctx,
			chromedp.Navigate(url),
			chromedp.OuterHTML(`#b_results`, &body, chromedp.ByID),
		)
	}

	if err != nil {
		return Response{}, err
	}
	return getBindResponseData(body, domain, pageIndex)
}
func getGoogleData(kwd, domain string, ctx context.Context, url string, pageIndex int) (Response, error) {
	body := ""
	next := ""
	var err error
	if pageIndex == 1 {
		err = chromedp.Run(ctx,
			chromedp.Navigate(`https://www.google.com`),
			chromedp.WaitVisible(`input[name="q"]`),
			chromedp.SendKeys(`input[name="q"]`, kwd, chromedp.ByQuery),
			chromedp.Click(`input[type="submit"]`, chromedp.ByQuery),
			chromedp.OuterHTML(`#rso`, &body, chromedp.ByID),
			chromedp.OuterHTML(`#pnnext`, &next, chromedp.ByID),
		)
	} else {
		err = chromedp.Run(ctx,
			chromedp.Navigate(url),
			chromedp.OuterHTML(`#rso`, &body, chromedp.ByID),
			chromedp.OuterHTML(`#pnnext`, &next, chromedp.ByID),

		)
	}
	if err != nil {
		return Response{}, err
	}
	return getGoogleResponseData(body, domain, next, pageIndex)
}
func getGoogleResponseData(body, domain, next string, pageIndex int) (Response, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))

	if err != nil {
		return Response{}, err
	}
	reg, _ := regexp.Compile(domain)
	index := 0
	doc.Find(".TbwUpd").Each(func(i int, selection *goquery.Selection) {
		arr := reg.FindAllStringSubmatch(selection.Text(), -1)
		if index == 0 && len(arr) > 0 {
			index = i + 1
		}
	})
	if index > 0 {
		return Response{
			Page:  pageIndex,
			Index: index,
		}, nil
	}
	doc, err = goquery.NewDocumentFromReader(strings.NewReader(next))
	if err != nil {
		return Response{}, err
	}
	nextUrl, _ := doc.Find("#pnnext").Attr("href")
	return Response{
		NextUrl: fmt.Sprintf("https://www.google.com%s", nextUrl),
	}, nil
}
