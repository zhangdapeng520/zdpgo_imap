package zdpgo_imap

/*
@Time : 2022/5/24 20:07
@Author : 张大鹏
@File : config.go
@Software: Goland2021.3.1
@Description:
*/

type Config struct {
	Debug       bool   `json:"debug"`
	LogFilePath string `json:"log_file_path"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	TmpDir      string `json:"tmp_dir"`
}
