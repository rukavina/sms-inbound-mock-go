package main

//Enums
const (
	MsgTypeText        = "text"
	MsgTypeWsi         = "wsi"
	MsgTypeResponse    = "response"
	MsgTypeDlr         = "dlr"
	MsgDirectionMO     = "MO"
	MsgDirectionMT     = "MT"
	MsgDirectionMTPUSH = "MT-PUSH"
	MsgDSCGSM          = "GSM"
	MsgDSCUCS          = "UCS"
	MsgDSCBINARY       = "BINARY"
)

//MOMessage is original MO
type MOMessage struct {
	MsgType   string  `json:"type"`
	MsgID     string  `json:"msgId"`
	Direction string  `json:"direction"`
	Operator  string  `json:"operator"`
	Sender    string  `json:"sender"`
	Receiver  string  `json:"receiver"`
	DSC       string  `json:"dsc"`
	Text      string  `json:"text"`
	AddOns    AddOns  `json:"addOns,omitempty"`
	Service   Service `json:"service"`
}

//Service data about premium sms service (keyword, application, session, MO msgId, etc.)
type Service struct {
	ServiceID       string `json:"serviceId"`
	MOMsgID         string `json:"moMsgId,omitempty"`
	BillInfo        string `json:"billInfo,omitempty"`
	Country         string `json:"country,omitempty"`
	TextServiceHead string `json:"textServiceHead,omitempty"`
	TextTail        string `json:"textTail,omitempty"`
}

//AddOns is other, non-standard or rare message attributes, mostly used internally
type AddOns struct {
	Language string `json:"language,omitempty"`
}

//Auth is authentication parameters (when needed)
type Auth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

//Billing is billing parameters
type Billing struct {
	Currency  string  `json:"currency"`
	Price     float64 `json:"price"`
	PriceCode string  `json:"priceCode,omitempty"`
}

//DlrRequest data for required dlr, including custom pass-through attributes
type DlrRequest struct {
	EventsMask  int               `json:"eventsMask,omitempty"`
	CallbackURL string            `json:"callbackUrl"`
	CustomData  map[string]string `json:"customData,omitempty"`
}

//MTRequest is MT request to server
type MTRequest struct {
	MsgType    string     `json:"type"`
	Direction  string     `json:"direction"`
	Operator   string     `json:"operator"`
	Sender     string     `json:"sender"`
	Receiver   string     `json:"receiver"`
	DSC        string     `json:"dsc"`
	Text       string     `json:"text"`
	Auth       Auth       `json:"auth"`
	DlrRequest DlrRequest `json:"dlrRequest"`
	Service    Service    `json:"service"`
	Billing    Billing    `json:"billing"`
}

//MTResponse is MT response from server
type MTResponse struct {
	MsgType   string `json:"type"`
	Direction string `json:"direction"`
	MsgID     string `json:"msgId"`
}

//MTDlr is MT dlr struct
type MTDlr struct {
	MsgType          string            `json:"type"`
	MsgID            string            `json:"msgId"`
	TotalMsgParts    int               `json:"totalMsgParts,omitempty"`
	MsgPart          int               `json:"msgPart,omitempty"`
	Operator         string            `json:"operator"`
	Sender           string            `json:"sender"`
	Receiver         string            `json:"receiver"`
	DlrCode          int               `json:"dlrCode"`
	DlrReason        string            `json:"dlrReason"`
	CustomData       map[string]string `json:"customData,omitempty"`
	CustomMask       int               `json:"customMask"`
	CustomURL        string            `json:"customUrl,omitempty"`
	EffectiveBilling EffectiveBilling  `json:"effectiveBilling"`
}

//EffectiveBilling is Information how actually certain message was billed (related with this particular event).
type EffectiveBilling struct {
	Currency string  `json:"currency"`
	Price    float64 `json:"price"`
	Kickback float64 `json:"kickback"`
}
