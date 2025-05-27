package SipHandlers

import "github.com/ghettovoice/gosip/sip"

type SipMsgUtilities interface {
	extractBranch(req sip.Request) string
	buildSipResponse(res sip.MessageID, req sip.Request, statusCode sip.StatusCode, reason string, body string, headers ...sip.Header) sip.Response
	validateRegisterUserNameMatchesAndExtract(req sip.Request) (string, sip.Response, error)
	respondToRequest(constructedResponse sip.Response, tx sip.ServerTransaction)
	extractIpAndPort(req sip.Request) (string, string, sip.Response, error)
}
