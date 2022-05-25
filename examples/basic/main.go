package main

import (
	"github.com/zhangdapeng520/zdpgo_imap"
)

/*
@Time : 2022/5/24 20:01
@Author : 张大鹏
@File : main.go
@Software: Goland2021.3.1
@Description:
*/

var (
	ic = zdpgo_imap.NewWithConfig(&zdpgo_imap.Config{
		Debug:    true,
		Username: "1156956636@qq.com",
		Password: "gluhsjysrosbbadc",
		Host:     "imap.qq.com",
		Port:     993,
	})
)

func main() {
	//ic.SearchByTitle("") // 搜索所有的邮件
	ic.SearchByRecent(10) // 搜索最近10封邮件

	if ic.Results != nil && len(ic.Results) > 0 {
		for _, result := range ic.Results {
			ic.Log.Debug("结果", "title", result.Title, "date", result.DateStr)
		}
	}
}
