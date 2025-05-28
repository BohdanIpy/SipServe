package SipHandlers

import "github.com/ghettovoice/gosip/sip"

type SipMsgUtilities interface {
	extractBranch(req sip.Request) (string, sip.Response, error)
	buildSipResponse(res sip.MessageID, req sip.Request, statusCode sip.StatusCode, reason string, body string, headers ...sip.Header) sip.Response
	validateRegisterUserNameMatchesAndExtract(req sip.Request) (string, sip.Response, error)
	respondToRequest(constructedResponse sip.Response, tx sip.ServerTransaction)
	extractIpAndPort(req sip.Request) (sip.Uri, sip.Response, error)
	extractCSeqNumber(req sip.Request) (uint32, sip.Response, error)
	extractCallUUID(req sip.Request) (string, sip.Response, error)
}
