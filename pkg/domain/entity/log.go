package entity

// HTTPMessage represents a complete HTTP message (request + response)
type HTTPMessage struct {
	Request  *HTTPRequest  `json:"request"`
	Response *HTTPResponse `json:"response"`
}

// HTTPRequest represents detailed HTTP request info
type HTTPRequest struct {
	Method        string            `json:"method"`
	URL           string            `json:"url"`
	Proto         string            `json:"proto"`
	Header        map[string]string `json:"header"`
	Body          string            `json:"body"`
	ContentLength int64             `json:"content_length"`
}

// HTTPResponse represents detailed HTTP response info
type HTTPResponse struct {
	Proto         string            `json:"proto"`
	StatusCode    int               `json:"status_code"`
	Status        string            `json:"status"`
	Header        map[string]string `json:"header"`
	Body          string            `json:"body"`
	ContentLength int64             `json:"content_length"`
}

// DNSMessage represents a complete DNS transaction
type DNSMessage struct {
	Domain   string     `json:"domain"`
	Server   string     `json:"server"`
	Request  *DNSDetail `json:"request"`
	Response *DNSDetail `json:"response"`
	RTT      int64      `json:"rtt"`   // in milliseconds
	Error    string     `json:"error"` // string error representation
}

// DNSDetail represents detailed DNS packet info
type DNSDetail struct {
	ID       uint16        `json:"id"`
	Response bool          `json:"response"`
	Opcode   int           `json:"opcode"`
	Rcode    int           `json:"rcode"`
	Question []DNSQuestion `json:"question"`
	Answer   []DNSRR       `json:"answer"`
	Nv       []DNSRR       `json:"authority"`
	Extra    []DNSRR       `json:"extra"`
}

// DNSQuestion represents a DNS question
type DNSQuestion struct {
	Name   string `json:"name"`
	Qtype  string `json:"qtype"`
	Qclass string `json:"qclass"`
}

// DNSRR represents a DNS resource record
type DNSRR struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Class string `json:"class"`
	TTL   uint32 `json:"ttl"`
	Data  string `json:"data"`
}
