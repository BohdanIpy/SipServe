package SipHandlers

import (
	"SipServe/MyHandlers"
	"context"
	"errors"
	"github.com/ghettovoice/gosip/log"
	"github.com/ghettovoice/gosip/sip"
	"github.com/redis/go-redis/v9"
	"github.com/satori/go.uuid"
)

type SipHandler struct {
	Logger log.Logger
	Client *redis.Client
	Ctx    context.Context
}

type ParseReturn struct {
	RegData RegisterData
	Resp    sip.Response
	Err     error
}

func (h *SipHandler) extractBranch(req sip.Request) (string, sip.Response, error) {
	via, ok := req.ViaHop()
	if !ok {
		constructedResponse := h.buildSipResponse(
			"",
			req,
			400,
			"Bad Request",
			"",
			&sip.GenericHeader{
				HeaderName: "Reason",
				Contents:   "Missing Via header",
			},
		)
		return "", constructedResponse, errors.New("missing Via header in request")
	}

	branch, exists := via.Params.Get("branch")
	if !exists {
		constructedResponse := h.buildSipResponse(
			"",
			req,
			400,
			"Bad Request",
			"",
			&sip.GenericHeader{
				HeaderName: "Reason",
				Contents:   "Missing branch in Via",
			},
		)
		return "", constructedResponse, errors.New("missing branch parameter in Via header")
	}
	return branch.String(), nil, nil
}

func (h *SipHandler) buildSipResponse(
	res sip.MessageID,
	req sip.Request,
	statusCode sip.StatusCode,
	reason string,
	body string,
	headers ...sip.Header,
) sip.Response {
	result := sip.NewResponseFromRequest(res, req, statusCode, reason, body)
	for _, header := range headers {
		if existing := result.GetHeaders(header.Name()); len(existing) > 0 {
			result.RemoveHeader(header.Name())
		}
		result.AppendHeader(header)
	}
	return result
}

// Used
func (h *SipHandler) validateRegisterUserNameMatchesAndExtract(req sip.Request) (string, sip.Response, error) {
	fromHeader, ok := req.From()
	if !ok {
		constructedResponse := h.buildSipResponse(
			"",
			req,
			400,
			"Bad Request",
			"",
			&sip.GenericHeader{
				HeaderName: "Reason",
				Contents:   "No username specified in From header",
			},
		)
		return "", constructedResponse, errors.New("no username specified in From header")
	}
	fromHeaderUsername := fromHeader.Address.User().String()
	toHeader, ok := req.To()
	if !ok {
		constructedResponse := h.buildSipResponse(
			"",
			req,
			400,
			"Bad Request",
			"",
			&sip.GenericHeader{
				HeaderName: "Reason",
				Contents:   "No username specified in To header",
			},
		)
		return "", constructedResponse, errors.New("no username specified in To header")
	}
	toHeaderUsername := toHeader.Address.User().String()
	contactHeader, ok := req.Contact()
	if !ok {
		constructedResponse := h.buildSipResponse(
			"",
			req,
			400,
			"Bad Request",
			"",
			&sip.GenericHeader{
				HeaderName: "Reason",
				Contents:   "No username specified in Contact header",
			},
		)
		return "", constructedResponse, errors.New("no username specified in Contact header")
	}
	contactHeaderUsername := contactHeader.Address.User().String()
	if fromHeaderUsername != toHeaderUsername ||
		toHeaderUsername != contactHeaderUsername ||
		fromHeaderUsername != contactHeaderUsername {
		constructedResponse := h.buildSipResponse(
			"",
			req,
			400,
			"Bad Request",
			"",
			&sip.GenericHeader{
				HeaderName: "Reason",
				Contents:   "Usernames do not match in headers To, From, Contact",
			},
		)
		return "", constructedResponse, errors.New("usernames do not match in headers To, From, Contact")

	}
	return fromHeaderUsername, nil, nil
}

func (h *SipHandler) respondToRequest(constructedResponse sip.Response, tx sip.ServerTransaction) {
	if err := tx.Respond(constructedResponse); err != nil {
		h.Logger.Errorf("Failed to respond: %v", err)
	}
}

func (h *SipHandler) extractIpAndPort(req sip.Request) (sip.Uri, sip.Response, error) {
	contact, success := req.Contact()
	if !success {
		constructedResponse := h.buildSipResponse("", req, 400, "Bad Request", "", &sip.GenericHeader{
			HeaderName: "Reason",
			Contents:   "Cannot extract the contact header",
		})
		return nil, constructedResponse, errors.New("error extracting contact header")
	}
	clientContactUriContact := contact.Address
	return clientContactUriContact, nil, nil
}

func (h *SipHandler) extractCSeqNumber(req sip.Request) (uint32, sip.Response, error) {
	cseq, success := req.CSeq()
	if !success {
		constructedResponse := h.buildSipResponse("", req, 400, "Bad Request", "", &sip.GenericHeader{
			HeaderName: "Reason",
			Contents:   "Cannot extract the cseq number",
		})
		return 0, constructedResponse, errors.New("error extracting cseq header")
	}
	num := cseq.SeqNo
	return num, nil, nil
}

func (h *SipHandler) extractCallUUID(req sip.Request) (string, sip.Response, error) {
	callid, success := req.CallID()
	if !success {
		constructedResponse := h.buildSipResponse("", req, 400, "Bad Request", "", &sip.GenericHeader{
			HeaderName: "Reason",
			Contents:   "Cannot extract the callid",
		})
		return "", constructedResponse, errors.New("cannot extract the callid")
	}
	return callid.Value(), nil, nil
}

func (h *SipHandler) extractFromTag(req sip.Request) (string, sip.Response, error) {
	fromHeader, success := req.From()
	if !success {
		constructedResponse := h.buildSipResponse("", req, 400, "Bad Request", "", &sip.GenericHeader{
			HeaderName: "Reason",
			Contents:   "Cannot extract the from",
		})
		return "", constructedResponse, errors.New("cannot extract the from header")
	}
	fromTag, success := fromHeader.Params.Get("tag")
	if !success {
		constructedResponse := h.buildSipResponse("", req, 400, "Bad Request", "", &sip.GenericHeader{
			HeaderName: "Reason",
			Contents:   "Cannot extract the tag from From header",
		})
		return "", constructedResponse, errors.New("cannot extract the tag from From header")
	}
	return fromTag.String(), nil, nil
}

func (h *SipHandler) parseRegistrationRequest(req sip.Request) (RegisterData, sip.Response, error) {
	uuidGenerated, err := uuid.NewV4()
	if err != nil {
		return RegisterData{}, nil, errors.New("failed to generate UUID")
	}

	// Constructing, adding the generated ToTag
	registerData := RegisterData{
		ToTag: uuidGenerated.String(),
	}

	// extracting the branch
	branch, response, err := h.extractBranch(req)
	if err != nil {
		return RegisterData{}, response, err
	}
	registerData.Branch = branch

	// extracting the username
	usernameString, response, err := h.validateRegisterUserNameMatchesAndExtract(req)
	if err != nil {
		return RegisterData{}, response, err
	}
	registerData.Username = usernameString

	// extracting the contactUri
	contactUri, response, err := h.extractIpAndPort(req)
	if err != nil {
		return RegisterData{}, response, err
	}
	registerData.ClientContactUri = contactUri

	// extracting CSeqNumber
	num, response, err := h.extractCSeqNumber(req)
	if err != nil {
		return RegisterData{}, response, err
	}
	registerData.CSeqNumber = num

	//  extracting the call id
	callid, response, err := h.extractCallUUID(req)
	if err != nil {
		return RegisterData{}, response, err
	}
	registerData.CallUUID = callid

	// extracting the `from tag`
	tagFrom, response, err := h.extractFromTag(req)
	if err != nil {
		return RegisterData{}, response, err
	}
	registerData.FromTag = tagFrom
	registerData.Content = req.Body()
	return registerData, nil, nil
}

func (h *SipHandler) HandleRegisterRequest(req sip.Request, tx sip.ServerTransaction) {
	h.Logger.Infof("Received SIP %s request for %s", req.Method(), req.Recipient())

	parsing := make(chan ParseReturn)
	go func() {
		parsed, response, err := h.parseRegistrationRequest(req)
		parsing <- ParseReturn{RegData: parsed, Resp: response, Err: err}
	}()

	result := <-parsing

	if result.Err != nil {
		h.Logger.Warnf("%s", result.Err.Error())
		err := tx.Respond(result.Resp)
		if err != nil {
			h.Logger.Warn("Failed to respond to the request")
		}
		return
	}
	registeredData := result.RegData

	err := MyHandlers.StoreOrUpdateUserInRedis(h.Ctx, h.Client,
		registeredData.Username,
		registeredData.ClientContactUri.Host(),
		registeredData.ClientContactUri.Port().String())
	if err != nil {
		h.Logger.Errorf("Failed to store user in Redis: %v", err)
	}

	res := h.buildSipResponse("", req, 200, "OK", "")

	h.Logger.Infof("SIP reply:\n%s", res.String())
	if err := tx.Respond(res); err != nil {
		h.Logger.Errorf("Failed to send SIP response: %v", err)
	}
}
