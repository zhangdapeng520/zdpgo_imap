package zdpgo_imap

import "testing"

func getImap() *Imap {
	ic := NewWithConfig(&Config{
		Debug:    true,
		Username: "1156956636@qq.com",
		Password: password,
		Host:     "imap.qq.com",
		Port:     993,
	})
	return ic
}

func TestImap_SearchByTitle(t *testing.T) {
	ic := getImap()
	_, err := ic.SearchByTitle("") // 搜索所有的邮件
	if err != nil {
		panic(err)
	}
}

func TestImap_SearchByRecent(t *testing.T) {
	ic := getImap()
	_, err := ic.SearchByRecent(10) // 搜索所有的邮件
	if err != nil {
		panic(err)
	}
}

func TestImap_SearchByContent(t *testing.T) {
	ic := getImap()
	_, err := ic.SearchByContent("ZkUrgkbvTGAMBwjT") // 搜索所有的邮件
	if err != nil {
		panic(err)
	}
}
