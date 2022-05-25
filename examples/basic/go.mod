module basic

go 1.17

require (
	github.com/emersion/go-imap v1.2.1
	github.com/emersion/go-message v0.15.0
	github.com/zhangdapeng520/zdpgo_imap v0.1.0
)

require (
	github.com/emersion/go-sasl v0.0.0-20200509203442-7bfe0ed36a21 // indirect
	github.com/emersion/go-textwrapper v0.0.0-20200911093747-65d896831594 // indirect
	github.com/zhangdapeng520/zdpgo_log v1.3.4 // indirect
	golang.org/x/text v0.3.7 // indirect
)

replace github.com/zhangdapeng520/zdpgo_imap v0.1.0 => ../../../zdpgo_imap
