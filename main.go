package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	target := flag.String("target", "", "目標 IP 或域名（必填）")
	site := flag.String("site", "", "結果頁網址，例如 https://mtr.example.com（必填）")
	flag.Parse()

	if *target == "" || *site == "" {
		fmt.Fprintln(os.Stderr, "用法: mtr-runner --target <host> --site <url>")
		os.Exit(1)
	}

	fmt.Printf("正在測試路由到 %s ...\n", *target)

	jsonOutput, err := runMtr(*target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "MTR 執行失敗：%v\n", err)
		os.Exit(1)
	}

	encoded, err := encodeData(jsonOutput)
	if err != nil {
		fmt.Fprintf(os.Stderr, "編碼失敗：%v\n", err)
		os.Exit(1)
	}

	resultURL := *site + "/result?data=" + encoded
	fmt.Println("完成！正在開啟瀏覽器...")

	if err := openBrowser(resultURL); err != nil {
		fmt.Printf("請手動開啟以下網址：\n%s\n", resultURL)
	}
}
