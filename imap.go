package zdpgo_imap

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"github.com/zhangdapeng520/zdpgo_imap/imap"
	"github.com/zhangdapeng520/zdpgo_imap/imap/client"
	"github.com/zhangdapeng520/zdpgo_imap/message/charset"
	"github.com/zhangdapeng520/zdpgo_imap/message/mail"
	"github.com/zhangdapeng520/zdpgo_log"
)

/*
@Time : 2022/5/24 20:07
@Author : 张大鹏
@File : imap.go
@Software: Goland2021.3.1
@Description:
*/

type Imap struct {
	Config         *Config
	Result         *Result
	Results        []*Result
	All            []*Result // 存储所有的邮件
	LastSearchTime time.Time // 最近一次搜索
	Client         *client.Client
	Log            *zdpgo_log.Log
}

func New() *Imap {
	return NewWithConfig(&Config{})
}

func NewWithConfig(config *Config) *Imap {
	i := &Imap{}

	// 日志
	if config.LogFilePath == "" {
		config.LogFilePath = "logs/zdpgo/zdpgo_imap.log"
	}
	i.Log = zdpgo_log.NewWithDebug(config.Debug, config.LogFilePath)

	// 配置
	if config.TmpDir == "" {
		config.TmpDir = ".zdpgo_imap_tmp_downloads"
	}
	i.Config = config

	return i
}

func (i *Imap) InitClient() {
	var err error

	// 【字符集】  处理us-ascii和utf-8以外的字符集(例如gbk,gb2313等)时, 需要加上这行代码。
	// 【参考】 https://github.com/zhangdapeng520/zdpgo_imap/imap/wiki/Charset-handling
	imap.CharsetReader = charset.Reader

	// 连接邮件服务器
	address := fmt.Sprintf("%s:%d", i.Config.Host, i.Config.Port)
	i.Client, err = client.DialTLS(address, nil)
	if err != nil {
		i.Log.Error("连接邮件服务器失败", "error", err)
		return
	}

	// 使用账号密码登录
	if err = i.Client.Login(i.Config.Username, i.Config.Password); err != nil {
		i.Log.Error("登录邮件服务器失败", "error", err, "config", i.Config)
	}
}

func (i *Imap) IsHealth() bool {
	var err error
	// 【字符集】  处理us-ascii和utf-8以外的字符集(例如gbk,gb2313等)时, 需要加上这行代码。
	// 【参考】 https://github.com/zhangdapeng520/zdpgo_imap/imap/wiki/Charset-handling
	imap.CharsetReader = charset.Reader

	// 连接邮件服务器
	address := fmt.Sprintf("%s:%d", i.Config.Host, i.Config.Port)
	i.Client, err = client.DialTLS(address, nil)
	if err != nil {
		i.Log.Error("连接邮件服务器失败", "error", err)
		return false
	}
	defer func(Client *client.Client) {
		err = Client.Close()
		if err != nil {
			i.Log.Error("关闭Imap客户端对象失败", "error", err)
		}
	}(i.Client)

	// 使用账号密码登录
	if err = i.Client.Login(i.Config.Username, i.Config.Password); err != nil {
		i.Log.Error("登录邮件服务器失败", "error", err, "config", i.Config)
		return true
	}

	return false
}

// SearchByTitle 根据邮件标题查询邮件
// 【处理业务需求】假设需求是找出求以subject开头的标题的最新邮件，并下载附件。
// 【思路】有些邮件包含附件后会变得特别大，如果要遍历的邮件很多，直接遍历处理，每封邮件都获取'RFC822'内容，
// fetch方法执行耗时可能会很长, 因此可以分两次fetch处理，减少处理时长：
// 1)第一次fetch先使用ENVELOP或者RFC822.HEADER获取邮件头信息找到满足业务需求邮件的id
// 2)第二次fetch根据这个邮件id使用'RFC822'获取邮件MIME内容，下载附件
func (i *Imap) SearchByTitle(title string) ([]*Result, error) {
	// 如果距离最近一次搜索不超过30秒钟，则使用最近搜索的数据
	now := time.Now()
	diffTime := now.Sub(i.LastSearchTime).Seconds()
	if diffTime < 10 {
		return i.All, nil
	}

	// 连接邮件服务器
	i.InitClient()
	defer func(Client *client.Client) {
		err := Client.Logout()
		if err != nil {
			i.Log.Error("注销失败", "error", err)
		}
	}(i.Client)

	// 选择收件箱
	_, err := i.Client.Select("INBOX", false)
	if err != nil {
		i.Log.Error("选择收件箱失败", "error", err)
		return nil, err
	}

	// 搜索条件实例对象
	criteria := imap.NewSearchCriteria()

	// ALL是默认条件
	// See RFC 3501 section 6.4.4 for a list of searching criteria.
	criteria.WithoutFlags = []string{"ALL"}

	// 执行搜索，获取所有的ID
	ids, err := i.Client.Search(criteria)
	if err != nil {
		i.Log.Error("搜索邮件失败", "error", err)
		return nil, err
	}

	// 片段
	var section imap.BodySectionName

	// 遍历邮件，进行搜索
	var results []*Result
	for {
		if len(ids) == 0 {
			break
		}
		id := pop(&ids)            // 获取ID
		seqSet := new(imap.SeqSet) // 获取索引集合
		seqSet.AddNum(id)          // 索引集合添加ID

		// 查询邮件信息，不查询邮件的附件和内容
		chanMessage := make(chan *imap.Message, 1)
		go func() {
			// 第一次fetch, 只抓取邮件头，邮件标志，邮件大小等信息，执行速度快
			err = i.Client.Fetch(seqSet, []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchRFC822Size}, chanMessage)
			if err != nil {
				// 【实践经验】这里遇到过的err信息是：ENVELOPE doesn't contain 10 fields
				// 原因是对方发送的邮件格式不规范，解析失败
				// 相关的issue: https://github.com/zhangdapeng520/zdpgo_imap/imap/issues/143
				i.Log.Error("抓取邮件内容失败", "error", err, "seqSet", seqSet)
			}
		}()

		message := <-chanMessage
		if message == nil {
			i.Log.Warning("邮件服务器没有返回消息内容", "message", message)
			continue
		}

		// 如果包含要查找的标题，则进一步搜索内容
		if strings.Contains(message.Envelope.Subject, title) {
			// 查询邮件的内容和附件，不查询信息
			chanMsg := make(chan *imap.Message, 1)
			go func() {
				// 这里是第二次fetch, 获取邮件MIME内容
				err = i.Client.Fetch(seqSet, []imap.FetchItem{imap.FetchRFC822}, chanMsg)
				if err != nil {
					i.Log.Error("获取邮件MIME内容失败", "error", err, "seqSet", seqSet)
				}
			}()

			msg := <-chanMsg
			if msg == nil {
				i.Log.Error("返回的邮件消息为空", "msg", msg)
				return nil, err
			}

			sectionName := msg.GetBody(&section)
			if sectionName == nil {
				i.Log.Error("获取片段名称失败", "sectionName", sectionName)
				return nil, err
			}

			// 创建邮件阅读器
			mailReader, err1 := mail.CreateReader(sectionName)
			if err1 != nil {
				i.Log.Error("创建邮件阅读器失败", "error", err)
				return nil, err1
			}

			// 设置邮件查询结果
			result, err := i.GetResult(message, mailReader)
			if err != nil {
				i.Log.Error("设置查询结果失败", "error", err)
				return nil, err
			}
			results = append(results, result)
		}
	}

	// 同步到全部结果
	i.LastSearchTime = time.Now()
	i.All = results
	return results, nil
}

// SearchByContent 根据内容搜索
func (i *Imap) SearchByContent(searchContent string) ([]*Result, error) {
	results, err := i.SearchByTitle("") // 搜索所有的邮件
	if err != nil {
		i.Log.Error("搜索邮件失败", "error", err)
		return nil, err
	}
	return results, nil
}

// 搜索最近的指定数量的邮件
func (i *Imap) SearchByRecent(recentNum uint32) ([]*Result, error) {
	// 连接邮件服务器
	i.InitClient()
	defer func(Client *client.Client) {
		err := Client.Logout()
		if err != nil {
			i.Log.Error("关闭邮件IMAP客户端失败", "error", err)
		}
	}(i.Client)

	// 选择收件箱
	mbox, err := i.Client.Select("INBOX", false)
	if err != nil {
		i.Log.Error("获取收件箱失败", "error", err)
		return nil, err
	}

	// 获取近指定数量封邮件
	from := recentNum
	to := mbox.Messages
	if mbox.Messages > recentNum {
		from = mbox.Messages - recentNum
	}
	seqSet := new(imap.SeqSet) // 索引集合
	seqSet.AddRange(from, to)  // 设置邮件搜索范围

	// 执行查询
	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		// 抓取邮件消息体传入到messages信道
		searchItems := []imap.FetchItem{
			imap.FetchEnvelope,     // 邮件信息
			imap.FetchInternalDate, // 时间
		}
		done <- i.Client.Fetch(seqSet, searchItems, messages)
	}()

	// 处理查询结果
	var results []*Result
	for msg := range messages {
		result := i.GetBasicResult(msg)
		results = append(results, result)
	}

	if err = <-done; err != nil {
		i.Log.Error("执行查询失败", "error", err)
		return nil, err
	}

	// 返回结果
	return results, nil
}

// GetBasicResult 获取基本结果信息
func (i *Imap) GetBasicResult(message *imap.Message) *Result {
	result := &Result{
		Title:    message.Envelope.Subject,
		SeqNum:   message.SeqNum,
		Size:     message.Size,
		Flags:    message.Flags,
		DateTime: message.InternalDate,
		Date:     int(message.InternalDate.Unix()),
		DateStr:  message.InternalDate.Format("2006-01-02 15:04:05"),
	}

	// 发件人
	for _, from := range message.Envelope.From {
		result.From = from.Address()
		break
	}

	// 收件人
	var toEmails []string
	for _, to := range message.Envelope.To {
		toEmails = append(toEmails, to.Address())
	}
	result.ToEmails = toEmails

	// 抄送
	var ccEmails []string
	for _, to := range message.Envelope.Cc {
		ccEmails = append(ccEmails, to.Address())
	}
	result.CcEmails = ccEmails

	// 密送
	var bccEmails []string
	for _, to := range message.Envelope.Bcc {
		bccEmails = append(bccEmails, to.Address())
	}
	result.BccEmails = bccEmails

	// 返回
	return result
}

// GetResult 获取查询结果
func (i *Imap) GetResult(message *imap.Message,
	mailReader *mail.Reader) (*Result, error) {
	var (
		err      error
		part     *mail.Part
		body     []byte
		filename string
	)

	result := i.GetBasicResult(message)

	// 处理消息体的每个part
	for {
		part, err = mailReader.NextPart()
		if err != nil {
			break
		}

		// 分别处理
		switch h := part.Header.(type) {
		case *mail.Header:
			// 获取请求头信息
			result.Key = h.Get("X-ZdpgoEmail-Auther")
		case *mail.InlineHeader:
			// 获取正文内容, text或者html
			body, err = ioutil.ReadAll(part.Body)
			if err != nil {
				i.Log.Error("获取正文内容失败", "error", err)
				return nil, err
			}
			result.Body = string(body)
		case *mail.AttachmentHeader:
			// 下载附件
			filename, err = h.Filename()
			if err != nil {
				i.Log.Error("获取附件名称失败", "error", err)
				return nil, err
			}
			if filename != "" {
				body, err = ioutil.ReadAll(part.Body)
				if err != nil && err != io.EOF {
					i.Log.Warning("读取附件内容失败", "error", err, "filename", filename, "body", body)
				}
				result.Attachments = append(result.Attachments, map[string][]byte{
					filename: body,
				})
			}
		}
	}

	// 返回
	return result, nil
}

func pop(list *[]uint32) uint32 {
	length := len(*list)
	lastEle := (*list)[length-1]
	*list = (*list)[:length-1]
	return lastEle
}
