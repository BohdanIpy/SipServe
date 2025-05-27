package SipHandlers

import "github.com/ghettovoice/gosip/sip"

type SipMsgHandlersInterface interface {
	handleRegisterRequest(req sip.Request, tx sip.ServerTransaction)
}
