package SipHandlers

import "github.com/ghettovoice/gosip/sip"

type SipMsgHandlersInterface interface {
	handleRegisterRequest(req sip.Request, tx sip.ServerTransaction)
	parseRegistrationRequest(req sip.Request) (RegisterData, sip.Response, error)
}
