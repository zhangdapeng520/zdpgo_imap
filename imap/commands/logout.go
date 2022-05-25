package commands

import (
	"github.com/zhangdapeng520/zdpgo_imap/imap"
)

// Logout is a LOGOUT command, as defined in RFC 3501 section 6.1.3.
type Logout struct{}

func (c *Logout) Command() *imap.Command {
	return &imap.Command{
		Name: "LOGOUT",
	}
}

func (c *Logout) Parse(fields []interface{}) error {
	return nil
}
