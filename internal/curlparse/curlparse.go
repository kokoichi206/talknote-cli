// Package curlparse は Talknote の「Copy as cURL」からブラウザセッションを抽出する。
package curlparse

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

type Result struct {
	Origin string
	SID    string
}

var (
	requestURLPattern = regexp.MustCompile(`https://[A-Za-z0-9.-]+\.company\.talknote\.com/[^'"\s\\]*`)
	sidPattern        = regexp.MustCompile(`(?:^|[;\s'"])TALKNOTE_SID2=([^;'"\s\\]+)`)
)

func Parse(curlText string) (Result, error) {
	requestURL := requestURLPattern.FindString(curlText)
	if requestURL == "" {
		return Result{}, fmt.Errorf("talknote の *.company.talknote.com URL が cURL にありません")
	}
	parsedURL, err := url.Parse(requestURL)
	if err != nil {
		return Result{}, fmt.Errorf("talknote URL を解析できません: %w", err)
	}
	match := sidPattern.FindStringSubmatch(curlText)
	if match == nil {
		return Result{}, fmt.Errorf("TALKNOTE_SID2 Cookie が cURL にありません（Cookie を含む Copy as cURL を使用してください）")
	}
	sid := match[1]
	if strings.ContainsAny(sid, ";, \t\r\n") {
		return Result{}, fmt.Errorf("TALKNOTE_SID2 Cookie に不正な区切り文字があります")
	}
	return Result{Origin: parsedURL.Scheme + "://" + parsedURL.Host, SID: sid}, nil
}
