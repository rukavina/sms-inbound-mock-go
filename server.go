package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

//Server is main http/ws handler logic
type Server struct {
	Hub *WSHub
}

func (s *Server) getUUID() string {
	uuid, err := exec.Command("uuidgen").Output()
	if err != nil {
		log.Fatal(err)
	}

	return strings.TrimSpace(string(uuid))
}

//MTErrorResponse generates MTResponse for error cases
func (s *Server) MTErrorResponse(errorCode string, message string) MTResponse {
	return MTResponse{}
}

func (s *Server) jsonResult(w http.ResponseWriter, httpCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpCode)
	json.NewEncoder(w).Encode(data)
}

//serveMT handles inbound gate MT requests
func (s *Server) serveMT(w http.ResponseWriter, r *http.Request) {
	//dump, err := httputil.DumpRequest(r, true)
	body, err := ioutil.ReadAll(r.Body)
	log.Print("MT server request raw body: ", string(body))
	var req MTRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		log.Printf("MT request invalid: %s", err)
		s.jsonResult(w, 420, s.MTErrorResponse("5", "Format of text/content parameter iswrong."))
		return
	}
	//check params
	if req.Auth.Username == "" || req.Text == "" || req.Operator == "" || req.Receiver == "" || req.Sender == "" {
		log.Println("MT request invalid params")
		s.jsonResult(w, 420, s.MTErrorResponse("110", "Mandatory parameter(s) is missing"))
		return
	}
	//send WS
	wsMsg := &WSMessage{
		MsgType: WSMsgTypeMT,
		Data: map[string]string{
			"type":                    req.MsgType,
			"direction":               req.Direction,
			"operator":                req.Operator,
			"sender":                  req.Sender,
			"receiver":                req.Receiver,
			"dsc":                     req.DSC,
			"text":                    req.Text,
			"auth.Username":           req.Auth.Username,
			"auth.Password":           req.Auth.Password,
			"dlrRequest.callbackURL":  req.DlrRequest.CallbackURL,
			"dlrRequest.eventsMask":   fmt.Sprintf("%d", req.DlrRequest.EventsMask),
			"service.serviceID":       req.Service.ServiceID,
			"service.country":         req.Service.Country,
			"service.moMsgID":         req.Service.MOMsgID,
			"service.textTail":        req.Service.TextTail,
			"service.textServiceHead": req.Service.TextServiceHead,
			"billing.currency":        req.Billing.Currency,
			"billing.price":           fmt.Sprintf("%f", req.Billing.Price),
			"billing.priceCode":       req.Billing.PriceCode,
		},
	}
	s.Hub.BroadcastMessage(wsMsg)

	messageID := s.getUUID()

	res := MTResponse{
		MsgType:   MsgTypeResponse,
		Direction: MsgDirectionMT,
		MsgID:     messageID,
	}
	s.jsonResult(w, 202, res)

	log.Printf("Valid MT request and replied: OK [%s] %v\n", messageID, res)

	if req.DlrRequest.CallbackURL == "" {
		return
	}

	dlr := MTDlr{
		MsgType:   MsgTypeDlr,
		MsgID:     messageID,
		Operator:  req.Operator,
		Sender:    req.Sender,
		Receiver:  req.Receiver,
		DlrCode:   1,
		DlrReason: "DELIVERED",
		EffectiveBilling: EffectiveBilling{
			Currency: req.Billing.Currency,
			Price:    req.Billing.Price,
			Kickback: 1.10,
		},
	}

	if req.DlrRequest.CustomData != nil {
		dlr.CustomData = req.DlrRequest.CustomData
	}

	//send dlr as go routine
	go s.sendDlr(req, dlr)
}

// send dlr
func (s *Server) sendDlr(req MTRequest, dlr MTDlr) {
	log.Println("Sending DLR notification to ", req.DlrRequest.CallbackURL)
	//give a timeout
	time.Sleep(time.Second * 2)
	res, err := s.makeHTTPRequest("POST", req.DlrRequest.CallbackURL, dlr)
	if err != nil {
		log.Printf("DLR response error %s", err)
		return
	}
	log.Printf("DLR response: %s", res)
}

// serveWs handles websocket requests from the peer.
func (s *Server) serveWs(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf("New client registering with IP [%s]\n", r.RemoteAddr)
	s.Hub.NewConnection(conn)
}

//OnNewWsMessage is hook to process incoming ws messages
func (s *Server) OnNewWsMessage(message *WSMessage) {
	if message.MsgType != WSMsgTypeMO {
		return
	}
	req := s.createMoMessage(message)
	if req == nil {
		log.Printf("Error unmarshal MO Message")
		return
	}
	url, ok := message.Data["url"]
	if !ok {
		log.Printf("MO URL not defined")
		return
	}
	s.sendMO(url, req)

}

//createMoMessage creates MO from WS request
func (s *Server) createMoMessage(message *WSMessage) *MOMessage {
	msgID := s.getUUID()
	req := &MOMessage{
		MsgType:   MsgTypeText,
		MsgID:     msgID,
		Direction: MsgDirectionMO,
		Operator:  message.Data["provider"],
		Sender:    message.Data["from"],
		Receiver:  message.Data["short_id"],
		DSC:       MsgDSCGSM,
		Text:      message.Data["text"],
		Service: Service{
			ServiceID: message.Data["sms_service_id"],
			MOMsgID:   msgID,
			Country:   message.Data["country"],
		},
		AddOns: AddOns{
			Language: message.Data["language"],
		},
	}
	sepIdx := strings.Index(req.Text, " ")
	if sepIdx > 0 {
		req.Service.TextServiceHead = req.Text[:sepIdx+1]
		req.Service.TextTail = req.Text[sepIdx+1:]
	}
	return req
}

// send MO
func (s *Server) sendMO(url string, req *MOMessage) {
	log.Printf("Sending MO notification to [%s]", url)

	res, err := s.makeHTTPRequest("POST", url, req)
	if err != nil {
		log.Printf("MO response error %s", err)
		return
	}
	log.Printf("MO response: %s", res)

	wsMsg := &WSMessage{
		MsgType: WSMsgTypeMORep,
		Data: map[string]string{
			"status": "success",
		},
	}
	s.Hub.BroadcastMessage(wsMsg)
}

//makeHTTPRequest returns needed params for http request from call and yate message
func (s *Server) makeHTTPRequest(method string, url string, req interface{}) (string, error) {
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	r, err := http.NewRequest(method, url, bytes.NewBuffer(reqBytes))
	r.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		log.Printf("HTTP request error: %s", err)
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Reponse status not OK/200 but [%d]", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	log.Printf("Client response: %s", string(body))
	return string(body), nil
}
