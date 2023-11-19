package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

const HostURL string = "https://ccp.netcup.net/run/webservice/servers/endpoint.php?JSON"

type CCPClient struct {
	hostURL    string
	httpClient http.Client
	authData   AuthData
	UserAgent  string
	DnsRecordsByDomain map[string][]DnsRecord
}

type AuthData struct {
	CustomerNumber string `json:"customernumber"`
	APIKey         string `json:"apikey"`
	SessionId      string `json:"apisessionid"`
}

type LoginData struct {
	CustomerNumber string `json:"customernumber"`
	APIKey         string `json:"apikey"`
	APIPassword    string `json:"apipassword"`
}

type RequestBody struct {
	Action string      `json:"action"`
	Param  interface{} `json:"param"`
}

type ResponseBody struct {
	ServerRequestId string `json:"serverrequestid"`
	Action          string `json:"action"`
	Status          string `json:"status"`     // Status of the Message like "error", "started", "pending", "warning" or "success".
	StatusCode      int    `json:"statuscode"` // Status code of the Message like 2011.
	ShortMessage    string `json:"shortmessage"`
	LongMessage     string `json:"longmessage"`
}

type SessionData struct {
	SessionId string `json:"apisessionid"`
}

type LoginResponse struct {
	ResponseBody
	ResponseData SessionData `json:"responsedata"`
}

type DomainInfoRequest struct {
	AuthData
	DomainName string `json:"domainname"`
}

type DnsZoneResponse struct {
	ResponseBody
	ResponseData DnsZone `json:"responsedata"`
}

type CreateDnsRecordsRequest struct {
	DomainInfoRequest
	DnsRecordSet NewDnsRecordSet `json:"dnsrecordset"`
}

type UpdateDnsRecordsRequest struct {
	DomainInfoRequest
	DnsRecordSet DnsRecordSet `json:"dnsrecordset"`
}

type DnsRecordsResponse struct {
	ResponseBody
	ResponseData DnsRecordSet `json:"responsedata"`
}

func NewCCPClient(customerNumber, apiKey, apiPassword string) (*CCPClient, error) {
	c := CCPClient{
		hostURL:    HostURL,
		httpClient: http.Client{Timeout: 10 * time.Second},
		DnsRecordsByDomain: make(map[string][]DnsRecord),
	}

	err := c.login(customerNumber, apiKey, apiPassword)

	if err != nil {
		return nil, err
	}

	return &c, nil
}

func (c *CCPClient) login(customerNumber, apiKey, apiPassword string) error {
	body, err := c.doRequest("login", LoginData{
		CustomerNumber: customerNumber,
		APIKey:         apiKey,
		APIPassword:    apiPassword,
	})
	res := LoginResponse{}
	err = json.Unmarshal(body, &res)
	if err != nil {
		return err
	}

	c.authData = AuthData{
		CustomerNumber: customerNumber,
		APIKey:         apiKey,
		SessionId:      res.ResponseData.SessionId,
	}
	return nil
}

func (c *CCPClient) doRequest(action string, param interface{}) ([]byte, error) {
	rb, err := json.Marshal(RequestBody{
		Action: action,
		Param:  param,
	})

	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.hostURL, strings.NewReader(string(rb)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status: %d, body: %s", res.StatusCode, body)
	}

	return body, err
}

func (c *CCPClient) GetDnsZone(domainName string) (*DnsZone, error) {
	body, err := c.doRequest("infoDnsZone", DomainInfoRequest{
		AuthData:   c.authData,
		DomainName: domainName,
	})

	if err != nil {
		return nil, err
	}

	res := DnsZoneResponse{}
	err = json.Unmarshal(body, &res)
	if err != nil {
		return nil, err
	}
	return &res.ResponseData, nil
}

func (c *CCPClient) GetDnsRecords(domainName string) ([]DnsRecord, error) {
	// check if we have the records for this domain cached to avoid triggering API rate limits
	records, present := c.DnsRecordsByDomain[domainName]
	if present {
		return records, nil
	}

	body, err := c.doRequest("infoDnsRecords", DomainInfoRequest{
		AuthData:   c.authData,
		DomainName: domainName,
	})
	fmt.Printf(string(body))

	if err != nil {
		return nil, err
	}

	res := DnsRecordsResponse{}
	err = json.Unmarshal(body, &res)
	if err != nil {
		return nil, err
	}

	// cache records for this domain
	c.DnsRecordsByDomain[domainName] = res.ResponseData.DnsRecords

	return res.ResponseData.DnsRecords, nil
}

func (c *CCPClient) GetDnsRecordById(domainName string, id string) (*DnsRecord, error) {
	records, err := c.GetDnsRecords(domainName)
	if err != nil {
		return nil, err
	}

	for _, record := range records {
		if record.Id == id {
			return &record, nil
		}
	}
	return nil, fmt.Errorf("could not find DNS record with ID %s for domain %s", id, domainName)
}

func (c *CCPClient) CreateDnsRecord(domainName string, record NewDnsRecord) (*DnsRecord, error) {
	fmt.Printf("%+v", record)
	fmt.Println(domainName)

	// flush cache for this domain to be sure we're not faking an incorrect state
	delete(c.DnsRecordsByDomain, domainName)

	body, err := c.doRequest("updateDnsRecords", CreateDnsRecordsRequest{
		DomainInfoRequest: DomainInfoRequest{
			AuthData:   c.authData,
			DomainName: domainName,
		},
		DnsRecordSet: NewDnsRecordSet{DnsRecords: []NewDnsRecord{record}},
	})

	if err != nil {
		return nil, err
	}

	res := DnsRecordsResponse{}
	err = json.Unmarshal(body, &res)
	if err != nil {
		fmt.Printf(string(body))
		return nil, err
	}

	newRecord, err := findNewRecord(res.ResponseData.DnsRecords, record)
	if err != nil {
		return nil, err
	}

	return newRecord, nil
}

func (c *CCPClient) UpdateDnsRecord(domainName string, record DnsRecord) (*DnsRecord, error) {
	// flush cache for this domain to be sure we're not faking an incorrect state
	delete(c.DnsRecordsByDomain, domainName)

	body, err := c.doRequest("updateDnsRecords", UpdateDnsRecordsRequest{
		DomainInfoRequest: DomainInfoRequest{
			AuthData:   c.authData,
			DomainName: domainName,
		},
		DnsRecordSet: DnsRecordSet{DnsRecords: []DnsRecord{record}},
	})

	if err != nil {
		return nil, err
	}

	res := DnsRecordsResponse{}
	err = json.Unmarshal(body, &res)
	if err != nil {
		fmt.Printf(string(body))
		return nil, err
	}

	newRecord, err := findRecordById(res.ResponseData.DnsRecords, record.Id)
	if err != nil {
		return nil, err
	}

	return newRecord, nil
}

func (c *CCPClient) DeleteDnsRecord(domainName string, record DnsRecord) error {
	// flush cache for this domain to be sure we're not faking an incorrect state
	delete(c.DnsRecordsByDomain, domainName)

	deleteRecord := record
	deleteRecord.DeleteRecord = true
	body, err := c.doRequest("updateDnsRecords", UpdateDnsRecordsRequest{
		DomainInfoRequest: DomainInfoRequest{
			AuthData:   c.authData,
			DomainName: domainName,
		},
		DnsRecordSet: DnsRecordSet{DnsRecords: []DnsRecord{deleteRecord}},
	})

	fmt.Printf(string(body))

	if err != nil {
		return err
	}

	return nil
}

func findRecordById(records []DnsRecord, id string) (*DnsRecord, error) {
	for _, record := range records {
		if record.Id == id {
			return &record, nil
		}
	}
	return nil, fmt.Errorf("could not find DNS record with ID %s", id)
}

func findNewRecord(newRecords []DnsRecord, requestedRecord NewDnsRecord) (*DnsRecord, error) {
	for _, record := range newRecords {
		if requestedRecord.Matches(record) {
			return &record, nil
		}
	}
	return nil, errors.New("could not retrieve newly created DNS record")
}

func (r NewDnsRecord) Matches(r2 DnsRecord) bool {
	isMatch := r.Hostname == r2.Hostname && r.Type == r2.Type && r.Destination == r2.Destination

	if r.Priority != "" {
		isMatch = isMatch && (r.Priority == r2.Priority)
	}
	return isMatch
}
