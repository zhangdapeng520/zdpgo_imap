package zdpgo_imap

import "time"

/*
@Time : 2022/5/24 20:46
@Author : 张大鹏
@File : form.go
@Software: Goland2021.3.1
@Description:
*/

type Result struct {
	From        string              `json:"from"`
	ToEmails    []string            `json:"to_emails"`
	CcEmails    []string            `json:"cc_emails"`
	BccEmails   []string            `json:"bcc_emails"`
	Date        int                 `json:"date"`
	DateStr     string              `json:"date_str"`
	DateTime    time.Time           `json:"date_time"`
	Key         string              `json:"key"`
	Title       string              `json:"title"`
	Body        string              `json:"body"`
	Attachments []map[string][]byte `json:"attachments"`
	Size        uint32              `json:"size"`
	Flags       []string            `json:"flags"`
	SeqNum      uint32              `json:"seq_num"`
}
