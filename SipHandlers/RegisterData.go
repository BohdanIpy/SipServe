package SipHandlers

import (
	"github.com/ghettovoice/gosip/sip"
)

type RegisterData struct {
	CallUUID         string         // ++
	ClientContactUri sip.ContactUri // ++
	FromTag          string         //++
	ToTag            string         // automatically generated
	CSeqNumber       uint32         // ++
	Branch           string         // ++
	Username         string         // ++
	Content          string         // ++
}
