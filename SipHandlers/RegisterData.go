package SipHandlers

import (
	"github.com/ghettovoice/gosip/sip"
	uuid "github.com/satori/go.uuid"
)

type RegisterData struct {
	CallUUID         uuid.UUID
	ClientContactUri sip.ContactUri
	FromTag          string
	ToTag            string // automatically generated
	CSeqNumber       uint32
	Branch           string
	Username         string
	Content          string
}
