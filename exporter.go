package main

import (
	"fmt"
	"github.com/echocat/nsone_exporter/model"
	"github.com/echocat/nsone_exporter/utils"
	"github.com/prometheus/client_golang/prometheus"
	"strings"
	"sync"
	"time"
	"github.com/prometheus/common/log"
)

type NsoneExportSettings struct {
	UsageByHourFilter    *model.Regexp
	UsageByDayFilter     *model.Regexp
	UsageByMonthFilter   *model.Regexp

	UsageOfAccount       bool
	UsageOfZonesFilter   *model.Regexp
	UsageOfRecordsFilter *model.Regexp

	QpsOfAccount         bool
	QpsOfZonesFilter     *model.Regexp
	QpsOfRecordsFilter   *model.Regexp
}

type NsoneExporter struct {
	client         *model.Client
	settings       NsoneExportSettings
	workerPool     *utils.WorkerPool
	collectionLock sync.RWMutex
	pointsLock     sync.RWMutex

	up     prometheus.Gauge
	points map[string]*prometheus.GaugeVec
}

func NewNsoneExporter(accessToken string, timeout time.Duration, numberOfWorkers int, nsoneNumberOfConcurrentConnections int, settings NsoneExportSettings) *NsoneExporter {
	points := map[string]*prometheus.GaugeVec{}
	if settings.QpsOfAccount {
		appendGauge(&points, "qps_account", "Queries per second of whole account.")
	}
	if settings.QpsOfZonesFilter.HasValue() {
		appendGauge(&points, "qps_zones", "Queries per second of all zones.")
	}
	if settings.QpsOfRecordsFilter.HasValue() {
		appendGauge(&points, "qps_records", "Queries per second of all records.")
	}
	if settings.UsageOfAccount {
		appendUsages(&points, "usage_account", "Export usages of whole account ", settings)
	}
	if settings.UsageOfZonesFilter.HasValue() {
		appendUsages(&points, "usage_zones", "Export usages of all zones ", settings)
	}
	if settings.UsageOfRecordsFilter.HasValue() {
		appendUsages(&points, "usage_records", "Export usages of all records ", settings)
	}

	return &NsoneExporter{
		settings:   settings,
		client:     model.NewClient(accessToken, timeout, nsoneNumberOfConcurrentConnections),
		workerPool: utils.NewWorkerPool(numberOfWorkers, numberOfWorkers),
		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "up",
			Help:      "Was the NSONE instance query successful?",
		}),
		points: points,
	}
}

func appendUsages(to *map[string]*prometheus.GaugeVec, namePrefix string, helpPrefix string, settings NsoneExportSettings) {
	if settings.UsageByHourFilter.HasValue() {
		appendGauge(to, namePrefix+"_hourly", helpPrefix+"by hour.")
	}
	if settings.UsageByDayFilter.HasValue() {
		appendGauge(to, namePrefix+"_daily", helpPrefix+"by day.")
	}
	if settings.UsageByMonthFilter.HasValue() {
		appendGauge(to, namePrefix+"_monthly", helpPrefix+"by month.")
	}
}

func appendGauge(to *map[string]*prometheus.GaugeVec, name string, help string) {
	labels := []string{}
	if strings.HasSuffix(name, "_zones") || strings.Contains(name, "_zones_") {
		labels = []string{
			"zone",
		}
	}
	if strings.HasSuffix(name, "_records") || strings.Contains(name, "_records_") {
		labels = []string{
			"zone",
			"record",
			"recordType",
		}
	}
	(*to)[name] = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      name,
		Help:      help,
	}, labels)
}

// Describe describes all the metrics ever exported by the
// exporter. It implements prometheus.Collector.
func (instance *NsoneExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- instance.up.Desc()
	for _, gauge := range instance.points {
		gauge.Describe(ch)
	}

}

// Collect fetches the stats from configured nsone and
// delivers them as Prometheus metrics. It implements prometheus.Collector.
func (instance *NsoneExporter) Collect(ch chan<- prometheus.Metric) {
	instance.collectionLock.Lock() // To protect metrics from concurrent collects.
	defer instance.collectionLock.Unlock()

	start := time.Now()
	log.Info("Collecting...")

	for _, gauge := range instance.points {
		gauge.Reset()
	}

	zones, err := instance.client.GetZones(true)
	if err == nil {
		log.Infof("Found %d active zones.", len(*zones))

		futures := &utils.WorkerFutures{}
		instance.exportUsageIfRequired(zones, futures)
		instance.exportQpsIfRequired(zones, futures)

		log.Infof("%d tasks enqueued.", len(*futures))

		err = futures.Wait()

	}
	if err == nil {
		for _, point := range instance.points {
			point.Collect(ch)
		}
	}

	duration := time.Now().Sub(start)
	if err != nil {
		log.Errorf("Collecting... FAILED! (duration: %v) Got: %v", duration, err)
		instance.up.Set(0)
	} else {
		log.Infof("Collecting... DONE! (duration: %v)", duration)
		instance.up.Set(1)
	}
	instance.up.Collect(ch)
}

func (instance *NsoneExporter) exportUsageIfRequired(zones *model.Zones, registerAt *utils.WorkerFutures) {
	instance.exportAccountUsageIfRequired(registerAt)
	instance.exportZoneUsagesIfRequired(registerAt)
	instance.exportRecordUsagesIfRequired(zones, registerAt)
}

func (instance *NsoneExporter) exportAccountUsageIfRequired(registerAt *utils.WorkerFutures) {
	if instance.settings.UsageOfAccount {
		if instance.settings.UsageByHourFilter.MatchString("account") {
			registerAt.Submit(instance.workerPool, func() error {
				usage, err := instance.client.GetAccountUsage(model.P_HOURLY)
				if err != nil {
					return err
				}
				return instance.setPoint("usage_account_hourly", usage.Queries, "", "", model.RT_NONE)
			})
		}
		if instance.settings.UsageByDayFilter.MatchString("account") {
			registerAt.Submit(instance.workerPool, func() error {
				usage, err := instance.client.GetAccountUsage(model.P_DAILY)
				if err != nil {
					return err
				}
				return instance.setPoint("usage_account_daily", usage.Queries, "", "", model.RT_NONE)
			})
		}
		if instance.settings.UsageByMonthFilter.MatchString("account") {
			registerAt.Submit(instance.workerPool, func() error {
				usage, err := instance.client.GetAccountUsage(model.P_MONTHLY)
				if err != nil {
					return err
				}
				return instance.setPoint("usage_account_monthly", usage.Queries, "", "", model.RT_NONE)
			})
		}
	}
}

func (instance *NsoneExporter) exportZoneUsagesIfRequired(registerAt *utils.WorkerFutures) {
	if instance.settings.UsageOfZonesFilter.HasValue() {
		if instance.settings.UsageByHourFilter.HasValue() {
			registerAt.Submit(instance.workerPool, func() error {
				usages, err := instance.client.GetZonesUsage(model.P_HOURLY)
				if err != nil {
					return err
				}
				for _, usage := range *usages {
					if instance.settings.UsageByHourFilter.MatchString(usage.Zone) && instance.settings.UsageOfZonesFilter.MatchString(usage.Zone) {
						err = instance.setPoint("usage_zones_hourly", usage.Queries, usage.Zone, "", model.RT_NONE)
						if err != nil {
							return err
						}
					}
				}
				return nil
			})
		}
		if instance.settings.UsageByDayFilter.HasValue() {
			registerAt.Submit(instance.workerPool, func() error {
				usages, err := instance.client.GetZonesUsage(model.P_DAILY)
				if err != nil {
					return err
				}
				for _, usage := range *usages {
					if instance.settings.UsageByHourFilter.MatchString(usage.Zone) && instance.settings.UsageOfZonesFilter.MatchString(usage.Zone) {
						err = instance.setPoint("usage_zones_daily", usage.Queries, usage.Zone, "", model.RT_NONE)
						if err != nil {
							return err
						}
					}
				}
				return nil
			})
		}
		if instance.settings.UsageByMonthFilter.HasValue() {
			registerAt.Submit(instance.workerPool, func() error {
				usages, err := instance.client.GetZonesUsage(model.P_MONTHLY)
				if err != nil {
					return err
				}
				for _, usage := range *usages {
					if instance.settings.UsageByHourFilter.MatchString(usage.Zone) && instance.settings.UsageOfZonesFilter.MatchString(usage.Zone) {
						err = instance.setPoint("usage_zones_monthly", usage.Queries, usage.Zone, "", model.RT_NONE)
						if err != nil {
							return err
						}
					}
				}
				return nil
			})
		}
	}
}

func (instance *NsoneExporter) exportRecordUsagesIfRequired(zones *model.Zones, registerAt *utils.WorkerFutures) {
	if instance.settings.UsageOfRecordsFilter.HasValue() {
		for _, zone := range *zones {
			if len(zone.Link) <= 0 && instance.settings.UsageOfRecordsFilter.MatchString(zone.Name) {
				instance.exportRecordUsagesOfZoneIfRequired(zone, registerAt)
			}
		}
	}
}

func (instance *NsoneExporter) exportRecordUsagesOfZoneIfRequired(zone *model.Zone, registerAt *utils.WorkerFutures) {
	if instance.settings.UsageByHourFilter.MatchString(zone.Name) {
		registerAt.Submit(instance.workerPool, func() error {
			usages, err := instance.client.GetRecordsUsage(zone.Name, model.P_HOURLY)
			if err != nil {
				return err
			}
			for _, usage := range *usages {
				fullRecord := usage.Type.String() + " " + usage.Domain
				if instance.settings.UsageByHourFilter.MatchString(fullRecord) && instance.settings.UsageOfRecordsFilter.MatchString(fullRecord) {
					err = instance.setPoint("usage_records_hourly", usage.Queries, usage.Zone, usage.Domain, usage.Type)
					if err != nil {
						return err
					}
				}
			}
			return nil
		})
	}
	if instance.settings.UsageByDayFilter.MatchString(zone.Name) {
		registerAt.Submit(instance.workerPool, func() error {
			usages, err := instance.client.GetRecordsUsage(zone.Name, model.P_DAILY)
			if err != nil {
				return err
			}
			for _, usage := range *usages {
				fullRecord := usage.Type.String() + " " + usage.Domain
				if instance.settings.UsageByDayFilter.MatchString(fullRecord) && instance.settings.UsageOfRecordsFilter.MatchString(fullRecord) {
					err = instance.setPoint("usage_records_daily", usage.Queries, usage.Zone, usage.Domain, usage.Type)
					if err != nil {
						return err
					}
				}
			}
			return nil
		})
	}
	if instance.settings.UsageByMonthFilter.MatchString(zone.Name) {
		registerAt.Submit(instance.workerPool, func() error {
			usages, err := instance.client.GetRecordsUsage(zone.Name, model.P_MONTHLY)
			if err != nil {
				return err
			}
			for _, usage := range *usages {
				fullRecord := usage.Type.String() + " " + usage.Domain
				if instance.settings.UsageByMonthFilter.MatchString(fullRecord) && instance.settings.UsageOfRecordsFilter.MatchString(fullRecord) {
					err = instance.setPoint("usage_records_monthly", usage.Queries, usage.Zone, usage.Domain, usage.Type)
					if err != nil {
						return err
					}
				}
			}
			return nil
		})
	}
}

func (instance *NsoneExporter) exportQpsIfRequired(zones *model.Zones, registerAt *utils.WorkerFutures) {
	instance.exportAccountQpsIfRequired(registerAt)
	instance.exportZonesQpsIfRequired(zones, registerAt)
	instance.exportRecordsQpsIfRequired(zones, registerAt)
}

func (instance *NsoneExporter) exportAccountQpsIfRequired(registerAt *utils.WorkerFutures) {
	if instance.settings.QpsOfAccount {
		registerAt.Submit(instance.workerPool, func() error {
			qps, err := instance.client.GetAccountQps()
			if err != nil {
				return err
			}
			return instance.setPoint("qps_account", qps, "", "", model.RT_NONE)
		})
	}
}

func (instance *NsoneExporter) exportZonesQpsIfRequired(zones *model.Zones, registerAt *utils.WorkerFutures) {
	if instance.settings.QpsOfZonesFilter.HasValue() {
		for _, zone := range *zones {
			instance.exportZoneQpsIfRequired(zone, registerAt)
		}
	}
}

func (instance *NsoneExporter) exportZoneQpsIfRequired(zone *model.Zone, registerAt *utils.WorkerFutures) {
	if len(zone.Link) <= 0 && instance.settings.QpsOfZonesFilter.MatchString(zone.Name) {
		registerAt.Submit(instance.workerPool, func() error {
			qps, err := instance.client.GetZoneQps(zone.Name)
			if err != nil {
				return err
			}
			return instance.setPoint("qps_zones", qps, zone.Name, "", model.RT_NONE)
		})
	}
}

func (instance *NsoneExporter) exportRecordsQpsIfRequired(zones *model.Zones, registerAt *utils.WorkerFutures) {
	if instance.settings.QpsOfRecordsFilter.HasValue() {
		for _, zone := range *zones {
			if len(zone.Link) <= 0 && instance.settings.QpsOfRecordsFilter.MatchString(zone.Name) {
				for _, record := range zone.Records {
					instance.exportRecordQpsIfRequired(zone, record, registerAt)
				}
			}
		}
	}
}

func (instance *NsoneExporter) exportRecordQpsIfRequired(zone *model.Zone, record *model.Record, registerAt *utils.WorkerFutures) {
	if len(record.Link) <= 0 && instance.settings.QpsOfRecordsFilter.MatchString(record.Type.String() + " " + record.Name) {
		registerAt.Submit(instance.workerPool, func() error {
			qps, err := instance.client.GetRecordQps(zone.Name, record.Name, record.Type)
			if err != nil {
				return err
			}
			return instance.setPoint("qps_records", qps, zone.Name, record.Name, record.Type)
		})
	}
}

func (instance *NsoneExporter) setPoint(name string, value float64, zone string, record string, recordType model.RecordType) error {
	instance.pointsLock.Lock() // To protect metrics from concurrent sets on points.
	defer instance.pointsLock.Unlock()
	labels := prometheus.Labels{}
	if strings.HasSuffix(name, "_zones") || strings.Contains(name, "_zones_") {
		labels["zone"] = zone
	}
	if strings.HasSuffix(name, "_records") || strings.Contains(name, "_records_") {
		labels["zone"] = zone
		labels["record"] = record
		labels["recordType"] = recordType.String()
	}
	gaugeVec := instance.points[name]
	if gaugeVec == nil {
		return fmt.Errorf("Try to set point with name %s but it was not crated before.", name)
	}
	gauge, err := gaugeVec.GetMetricWith(labels)
	if err != nil {
		return fmt.Errorf("Try to set point %s but got: %v", name, err)
	}
	gauge.Set(value)
	return nil
}
