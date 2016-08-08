package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/echocat/nsone_exporter/utils"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
	"github.com/prometheus/common/log"
	"crypto/tls"
)

const apiRootUri = "https://api.nsone.net/v1"

type Client struct {
	uri                                  string
	accessToken                          string
	maximumNumberOfConcurrentConnections int
	numberOfActiveConnections            int
	condition                            *sync.Cond
	client                               *http.Client
}

func NewClient(accessToken string, timeout time.Duration, maximumNumberOfConcurrentConnections int) *Client {
	return &Client{
		accessToken: accessToken,
		maximumNumberOfConcurrentConnections: maximumNumberOfConcurrentConnections,
		numberOfActiveConnections: 0,
		condition:   &sync.Cond{L: &sync.Mutex{}},
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: utils.LoadInternalCaBunlde(),
				},
				Dial: func(netw, addr string) (net.Conn, error) {
					c, err := net.DialTimeout(netw, addr, timeout)
					if err != nil {
						return nil, err
					}
					if err := c.SetDeadline(time.Now().Add(timeout)); err != nil {
						return nil, err
					}
					return c, nil
				},
			},
		},
	}
}

func (instance *Client) GetZones(expandZones bool) (*Zones, error) {
	uri, err := instance.zonesUriFor("", "", RT_NONE)
	if err != nil {
		return nil, err
	}
	result := &Zones{}
	err = instance.executeAndEvaluateUri(uri, err, result)
	if err != nil {
		return nil, err
	}
	if expandZones {
		instance.expandZonesOf(result)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (instance *Client) GetZone(zone string) (*Zone, error) {
	uri, err := instance.zonesUriFor(zone, "", RT_NONE)
	result := &Zone{}
	err = instance.executeAndEvaluateUri(uri, err, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (instance *Client) GetRecord(zone string, record string, recordType RecordType) (*Record, error) {
	uri, err := instance.zonesUriFor(zone, record, recordType)
	result := &Record{}
	err = instance.executeAndEvaluateUri(uri, err, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (instance *Client) GetAccountUsage(period StatsPeriod) (*Usage, error) {
	uri, err := instance.usagesUriFor("", "", RT_NONE, false, period)
	result := &Usages{}
	err = instance.executeAndEvaluateUri(uri, err, result)
	if err != nil {
		return nil, err
	}
	if len(*result) != 1 {
		return nil, fmt.Errorf("Expected number of elements in usages array is 1 but got %d.", len(*result))
	}
	return (*result)[0], nil
}

func (instance *Client) GetZonesUsage(period StatsPeriod) (*Usages, error) {
	uri, err := instance.usagesUriFor("", "", RT_NONE, true, period)
	result := &Usages{}
	err = instance.executeAndEvaluateUri(uri, err, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (instance *Client) GetZoneUsage(zone string, period StatsPeriod) (*Usages, error) {
	uri, err := instance.usagesUriFor(zone, "", RT_NONE, false, period)
	result := &Usages{}
	err = instance.executeAndEvaluateUri(uri, err, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (instance *Client) GetRecordsUsage(zone string, period StatsPeriod) (*Usages, error) {
	uri, err := instance.usagesUriFor(zone, "", RT_NONE, true, period)
	result := &Usages{}
	err = instance.executeAndEvaluateUri(uri, err, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (instance *Client) GetRecordUsage(zone string, record string, recordType RecordType, period StatsPeriod) (*Usages, error) {
	uri, err := instance.usagesUriFor(zone, record, recordType, false, period)
	result := &Usages{}
	err = instance.executeAndEvaluateUri(uri, err, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (instance *Client) GetAccountQps() (float64, error) {
	uri, err := instance.qpsUriFor("", "", RT_NONE)
	result := &QpsStat{}
	err = instance.executeAndEvaluateUri(uri, err, result)
	if err != nil {
		return 0, err
	}
	return result.Qps, nil
}

func (instance *Client) GetZoneQps(zone string) (float64, error) {
	uri, err := instance.qpsUriFor(zone, "", RT_NONE)
	result := &QpsStat{}
	err = instance.executeAndEvaluateUri(uri, err, result)
	if err != nil {
		return 0, err
	}
	return result.Qps, nil
}

func (instance *Client) GetRecordQps(zone string, record string, recordType RecordType) (float64, error) {
	uri, err := instance.qpsUriFor(zone, record, recordType)
	result := &QpsStat{}
	err = instance.executeAndEvaluateUri(uri, err, result)
	if err != nil {
		return 0, err
	}
	return result.Qps, nil
}

func (instance *Client) expandZonesOf(zones *Zones) error {
	futures := utils.WorkerFutures{}
	for _, zone := range *zones {
		instance.submitExpandZone(zone, &futures)
	}
	err := futures.Wait()
	return err
}

func (instance *Client) submitExpandZone(zone *Zone, registerAt *utils.WorkerFutures) {
	future := utils.NewWorkerFutureFor(func() error {
		fullZone, err := instance.GetZone(zone.Name)
		if err != nil {
			return fmt.Errorf("Could not retreive detailed infomation about zone %v. Cause: %v", zone.Name, err)
		}
		zone.Records = fullZone.Records
		return nil
	})
	go future.Execute()
	registerAt.Append(future)
}

func (instance *Client) requestFor(url *url.URL) *http.Request {
	return &http.Request{
		Method:     "GET",
		URL:        url,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header: http.Header{
			"X-NSONE-Key": []string{instance.accessToken},
		},
		Host: url.Host,
	}
}

func (instance *Client) executeAndEvaluateUri(uri *url.URL, err error, target interface{}) error {
	if err != nil {
		return err
	}
	request := instance.requestFor(uri)
	return instance.executeAndEvaluateRequest(request, target)
}

func (instance *Client) executeAndEvaluateRequest(request *http.Request, target interface{}) error {
	instance.increaseUsageCount()
	defer instance.decreaseUsageCount()
	var response *http.Response
	var err error
	waitBeforeRetry := 0
	for i := 0; (i == 0 || waitBeforeRetry > 0) && i < 20; i++ {
		if waitBeforeRetry > 0 {
			time.Sleep(time.Duration(i * waitBeforeRetry) * time.Millisecond)
			waitBeforeRetry = 0
		}
		response, err = instance.client.Do(request)
		if hasTimeoutError(err) {
			waitBeforeRetry = 50
			log.Warnf("Got timeout error while execute %v. Slow down and retry...", request.URL)
		}
		if err == nil && response.StatusCode == 429 {
			waitBeforeRetry = 2000
			log.Warnf("Got 'Rate limit exceeded' error while execute %v. Slow down and retry...", request.URL)
		}
	}
	if err != nil {
		return fmt.Errorf("Could not execute request %v. Got: %v", request.URL, err)
	}
	if response.StatusCode == 404 {
		return notFoundError{URL: request.URL, Err: err}
	}
	if response.StatusCode < 200 || response.StatusCode >= 400 {
		return fmt.Errorf("Could not execute request %v. Got: %v - %v", request.URL, response.StatusCode, response.Status)
	}
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(target)
	if err != nil {
		return fmt.Errorf("Could not execute request %v. Could not decode response. Got: %v", request.URL, err)
	}
	return nil
}

type notFoundError struct {
	URL *url.URL
	Err error
}

func (instance notFoundError) Error() string {
	return fmt.Sprintf("Could not execute request: %v. Got: 404 not found.", instance.URL)
}

func hasTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if urlErr, ok := err.(*url.Error); ok {
		return urlErr.Timeout()
	}
	return false
}

func (instance *Client) increaseUsageCount() {
	instance.condition.L.Lock()
	defer instance.condition.L.Unlock()
	for instance.numberOfActiveConnections >= instance.maximumNumberOfConcurrentConnections {
		instance.condition.Wait()
	}
	instance.numberOfActiveConnections++
}


func (instance *Client) decreaseUsageCount() {
	instance.condition.L.Lock()
	defer instance.condition.L.Unlock()
	instance.numberOfActiveConnections--
	instance.condition.Broadcast()
}

func (instance *Client) zonesUriFor(zone string, record string, recordType RecordType) (*url.URL, error) {
	uri := fmt.Sprintf("%s/zones", apiRootUri)
	if zone != "" {
		uri += fmt.Sprintf("/%s", zone)
		if record != "" {
			if recordType == RT_NONE {
				return nil, errors.New("It is not possible to provide a record without recordType.")
			}
			uri += fmt.Sprintf("/%s/%v", record, recordType)
		}
	} else if record != "" || recordType != RT_NONE {
		return nil, errors.New("It is not possible to provide a record and/or recordType without zone.")
	}
	result, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("Could not create zones uri for zone=%s, record=%s and type=%v. Cause: %v", zone, record, recordType, err)
	}
	return result, nil
}

func (instance *Client) usagesUriFor(zone string, record string, recordType RecordType, expand bool, period StatsPeriod) (*url.URL, error) {
	uri := fmt.Sprintf("%s/stats/usage", apiRootUri)
	if zone != "" {
		uri += fmt.Sprintf("/%s", zone)
		if record != "" {
			if recordType == RT_NONE {
				return nil, errors.New("It is not possible to provide a record without recordType.")
			}
			uri += fmt.Sprintf("/%s/%v", record, recordType)
		}
	} else if record != "" || recordType != RT_NONE {
		return nil, errors.New("It is not possible to provide a record and/or recordType without zone.")
	}
	uri += fmt.Sprintf("?period=%v&expand=%v", period, expand)
	result, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("Could not create usage uri for zone=%s, record=%s and type=%v. Cause: %v", zone, record, recordType, err)
	}
	return result, nil
}

func (instance *Client) qpsUriFor(zone string, record string, recordType RecordType) (*url.URL, error) {
	uri := fmt.Sprintf("%s/stats/qps", apiRootUri)
	if zone != "" {
		uri += fmt.Sprintf("/%s", zone)
		if record != "" {
			if recordType == RT_NONE {
				return nil, errors.New("It is not possible to provide a record without recordType.")
			}
			uri += fmt.Sprintf("/%s/%v", record, recordType)
		}
	} else if record != "" || recordType != RT_NONE {
		return nil, errors.New("It is not possible to provide a record and/or recordType without zone.")
	}
	result, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("Could not create usage uri for zone=%s, record=%s and type=%v. Cause: %v", zone, record, recordType, err)
	}
	return result, nil
}
