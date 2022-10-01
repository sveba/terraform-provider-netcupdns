package client

type DnsZone struct {
	Name         string `json:"domainname"`
	TTL          string `json:"ttl"`
	Serial       string `json:"serial"`
	Refresh      string `json:"refresh"`
	Retry        string `json:"retry"`
	Expire       string `json:"expire"`
	DNSSecStatus bool   `json:"dnssecstatus"`
}

type DnsRecord struct {
	Id           string `json:"id,omitempty"`
	Hostname     string `json:"hostname"`
	Type         string `json:"type"`
	Priority     string `json:"priority,omitempty"`
	Destination  string `json:"destination"`
	DeleteRecord bool   `json:"deleterecord,omitempty"`
	State        string `json:"state,omitempty"`
}

type NewDnsRecord struct {
	Hostname    string `json:"hostname"`
	Type        string `json:"type"`
	Priority    string `json:"priority,omitempty"`
	Destination string `json:"destination"`
}

type DnsRecordSet struct {
	DnsRecords []DnsRecord `json:"dnsrecords,omitempty"`
}

type NewDnsRecordSet struct {
	DnsRecords []NewDnsRecord `json:"dnsrecords"`
}
