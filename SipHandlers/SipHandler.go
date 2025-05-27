package SipHandlers

import (
	"errors"
	"fmt"
	"github.com/ghettovoice/gosip/log"
	"github.com/ghettovoice/gosip/sip"
	"github.com/satori/go.uuid"
	"strconv"
)

type SipHandler struct {
	Logger log.Logger
}

func (h *SipHandler) extractBranch(req sip.Request) string {
	via, ok := req.ViaHop()
	if !ok {
		h.Logger.Warn("No Via header found in request")
		return ""
	}

	branch, exists := via.Params.Get("branch")
	if !exists {
		h.Logger.Warn("Branch parameter does not set")
		return ""
	}

	return branch.String()
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

func (h *SipHandler) validateRegisterUserNameMatches(req sip.Request) (string, sip.Response, error) {
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

func (h *SipHandler) extractIpAndPort(req sip.Request) (string, string, sip.Response, error) {
	contactHeaders := req.GetHeaders("Contact")
	if len(contactHeaders) == 0 {
		h.Logger.Warn("missing Contact header")
		res := sip.NewResponseFromRequest("", req, 400, "Bad Request", "")
		res.AppendHeader(&sip.GenericHeader{
			HeaderName: "Reason",
			Contents:   "No contact specified in Headers",
		})
		tx.Respond(res)
		return
	}

	contact, ok := contactHeaders[0].(*sip.ContactHeader)
	if !ok {
		logger.Warn("invalid Contact header")
		res := sip.NewResponseFromRequest("", req, 400, "Bad Request", "")
		res.AppendHeader(&sip.GenericHeader{
			HeaderName: "Reason",
			Contents:   "invalid contact header",
		})
		tx.Respond(res)
		return
	}

	contactURI := contact.Address
	ip := contactURI.Host()
	port := "5060" // Default SIP port
	if contactURI.Port() != nil {
		port = fmt.Sprintf("%d", *contactURI.Port())
	}
	return "", "", nil
}

func (h *SipHandler) handleRegisterRequest(req sip.Request, tx sip.ServerTransaction) {
	h.Logger.Infof("Received SIP %s request for %s", req.Method(), req.Recipient())

	branch := h.extractBranch(req)
	if branch == "" {
		constructedResponse := h.buildSipResponse(
			"",
			req,
			400,
			"Bad Request",
			"",
			&sip.GenericHeader{
				HeaderName: "Reason",
				Contents:   "Missing branch parameter in Via header",
			},
		)
		h.respondToRequest(constructedResponse, tx)
		h.Logger.Warn("Missing branch parameter in Via header")
		return
	}

	// validating if name matches in all the request
	// extract the username
	username, constructedResponse, err := h.validateRegisterUserNameMatches(req)
	if err != nil {
		h.respondToRequest(constructedResponse, tx)
		return
	}

	// ip and port

	/*
		contactHeaders := req.GetHeaders("Contact")
			if len(contactHeaders) == 0 {
				logger.Warn("missing Contact header")
				res := sip.NewResponseFromRequest("", req, 400, "Bad Request", "")
				res.AppendHeader(&sip.GenericHeader{
					HeaderName: "Reason",
					Contents:   "No contact specified in Headers",
				})
				tx.Respond(res)
				return
			}

			contact, ok := contactHeaders[0].(*sip.ContactHeader)
			if !ok {
				logger.Warn("invalid Contact header")
				res := sip.NewResponseFromRequest("", req, 400, "Bad Request", "")
				res.AppendHeader(&sip.GenericHeader{
					HeaderName: "Reason",
					Contents:   "invalid contact header",
				})
				tx.Respond(res)
				return
			}

			contactURI := contact.Address
			ip := contactURI.Host()
			port := "5060" // Default SIP port
			if contactURI.Port() != nil {
				port = fmt.Sprintf("%d", *contactURI.Port())
			}
	*/

	// expires
	expires := 3600 // Default

	if hdr := req.GetHeaders("Expires"); hdr != nil {
		if expVal, ok := hdr[0].(*sip.Expires); ok {
			expires = int(*expVal)
		}
	} else if expParam, ok := contact.Params.Get("expires"); ok {
		expires, _ = strconv.Atoi(expParam.String())
	}

	err := StoreOrUpdateUserInRedis(username, ip, port, expires)
	if err != nil {
		logger.Errorf("Failed to store user in Redis: %v", err)
	}

	res := sip.NewResponseFromRequest("", req, 200, "OK", "")

	contactHeader, ok := req.Contact()
	if !ok {
		res := sip.NewResponseFromRequest("", req, 400, "Bad Request", "")
		res.AppendHeader(&sip.GenericHeader{
			HeaderName: "Reason",
			Contents:   "Missing Contact parameter",
		})
		tx.Respond(res)
		return
	}
	res.AppendHeader(contactHeader)

	toHeaders := req.GetHeaders("To")
	if len(toHeaders) == 0 {
		fmt.Println("No To headers found")
		return
	}

	oldTo, ok := toHeaders[0].(*sip.ToHeader)
	if !ok {
		fmt.Println("Failed to cast To header")
		return
	}

	newTag, _ := uuid.NewV4()
	newParams := sip.NewParams().Add("tag", sip.String{Str: newTag.String()})

	newToHeader := &sip.ToHeader{
		Address: oldTo.Address,
		Params:  newParams,
	}

	res.RemoveHeader("To")
	res.AppendHeader(newToHeader)

	logger.Infof("SIP reply:\n%s", res.String())

	if err := tx.Respond(res); err != nil {
		logger.Errorf("Failed to send SIP response: %v", err)
	}
}
