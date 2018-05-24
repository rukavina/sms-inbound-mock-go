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

//MOMessage is original MO
type MOMessage struct {
	ShortID   string `json:"short_id"`
	From      string `json:"from"`
	Text      string `json:"text"`
	Provider  string `json:"provider"`
	Keyword   string `json:"keyword"`
	MessageID string `json:"message_id"`
	Language  string `json:"language"`
}

//MOReply is expected reply from client to server
type MOReply struct {
	Status string `json:"status"`
}

//MTRequest is MT request to server
type MTRequest struct {
	Account  string `json:"account"`
	Username string `json:"username"`
	Password string `json:"password"`
	ShortID  string `json:"short_id"`
	To       string `json:"to"`
	Text     string `json:"text"`
	Provider string `json:"provider"`
	Keyword  string `json:"keyword"`
	Price    string `json:"price"`
	ExtID    string `json:"ext_id"`
	DlrURL   string `json:"dlr_url"`
}

//MTResponse is MT response from server
type MTResponse struct {
	MsgID     string `json:"msg_id"`
	Status    string `json:"status"`
	ErrorCode string `json:"error_code"`
	ErrorDesc string `json:"error_desc"`
}

//MTDlr is MT dlr struct
type MTDlr struct {
	Mobile  string `json:"mobile"`
	ShortID string `json:"short_id"`
	MsgID   string `json:"msgId"`
	ExtID   string `json:"ext_id"`
	Status  string `json:"status"`
	Price   string `json:"price"`
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
	return MTResponse{
		Status:    "error",
		ErrorCode: errorCode,
		ErrorDesc: message,
	}
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
		log.Panicf("MT request invalid: %s", err)
		s.jsonResult(w, 420, s.MTErrorResponse("5", "Format of text/content parameter iswrong."))
		return
	}
	//check params
	if req.Keyword == "" || req.Text == "" || req.Price == "" || req.Provider == "" || req.ShortID == "" || req.To == "" {
		log.Println("MT request invalid params")
		s.jsonResult(w, 420, s.MTErrorResponse("110", "Mandatory parameter(s) is missing"))
		return
	}
	//send WS
	var wsData map[string]string
	err = json.Unmarshal(body, &wsData)
	if err == nil {
		wsMsg := &WSMessage{
			MsgType: WSMsgTypeMT,
			Data:    wsData,
		}
		s.Hub.BroadcastMessage(wsMsg)
	} else {
		log.Printf("Error decoding to WsData: %s", err)
	}

	messageID := s.getUUID()

	res := MTResponse{
		MsgID:  messageID,
		Status: "success",
	}
	s.jsonResult(w, 202, res)

	log.Printf("Valid MT request and replied: OK [%s] %v\n", messageID, res)

	if req.DlrURL == "" {
		return
	}

	dlr := MTDlr{
		Mobile:  req.To,
		ShortID: req.ShortID,
		MsgID:   messageID,
		ExtID:   req.ExtID,
		Status:  "1",
		Price:   req.Price,
	}

	//send dlr as go routine
	go s.sendDlr(req, dlr)
}

// send dlr
func (s *Server) sendDlr(req MTRequest, dlr MTDlr) {
	log.Println("Sending DLR notification to ", req.DlrURL)
	//give a timeout
	time.Sleep(time.Second * 2)
	res, err := s.makeHTTPRequest("POST", req.DlrURL, dlr)
	if err != nil {
		log.Printf("DLR response error %s", err)
		return
	}
	log.Printf("DLR response: %v", res)
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
	data, _ := json.Marshal(message.Data)
	var req *MOMessage
	err := json.Unmarshal(data, &req)
	if err != nil {
		log.Printf("Error unmarshal MO Message: %s", err)
		return
	}
	req.MessageID = s.getUUID()
	words := strings.Split(req.Text, " ")
	req.Keyword = words[0] + "@" + req.ShortID
	url, ok := message.Data["url"]
	if !ok {
		log.Printf("MO URL not defined")
		return
	}
	s.sendMO(url, req)

}

// send MO
func (s *Server) sendMO(url string, req *MOMessage) {
	log.Printf("Sending MO notification to [%s]", url)

	res, err := s.makeHTTPRequest("POST", url, req)
	if err != nil {
		log.Printf("MO response error %s", err)
		return
	}
	log.Printf("MO response: %v", res)

	wsMsg := &WSMessage{
		MsgType: WSMsgTypeMORep,
		Data: map[string]string{
			"status": res["status"],
		},
	}
	s.Hub.BroadcastMessage(wsMsg)
}

//makeHTTPRequest returns needed params for http request from call and yate message
func (s *Server) makeHTTPRequest(method string, url string, req interface{}) (map[string]string, error) {
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	r, err := http.NewRequest(method, url, bytes.NewBuffer(reqBytes))
	r.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		log.Printf("HTTP request error: %s", err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Reponse status not OK/200 but [%d]", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	log.Printf("Client response: %s", string(body))
	var res map[string]string
	err = json.Unmarshal(body, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}
