package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"os"
	"strings"
	"time"
	"github.com/echocat/nsone_exporter/model"
)

const (
	namespace = "nsone"
)

var (
	name        = "nsone_exporter"
	version     = "devel"
	description = ""

	listenAddress = flag.String("web.listen-address", ":9113", "Address to listen on for web interface and telemetry.")
	metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
	tlsCert       = flag.String("web.tls-cert", "", "Path to PEM file that conains the certificate (and optionally also the private key in PEM format).\n"+
		"\tThis should include the whole certificate chain.\n"+
		"\tIf provided: The web socket will be a HTTPS socket.\n"+
		"\tIf not provided: Only HTTP.")
	tlsPrivateKey = flag.String("web.tls-private-key", "", "Path to PEM file that contains the private key (if not contained in web.tls-cert file).")
	tlsClientCa   = flag.String("web.tls-client-ca", "", "Path to PEM file that conains the CAs that are trused for client connections.\n"+
		"\tIf provided: Connecting clients should present a certificate signed by one of this CAs.\n"+
		"\tIf not provided: Every client will be accepted.")
	nsoneToken                         = flag.String("nsone.token", "", "Token to access the API of nsone.")
	nsoneTimeout                       = flag.Duration("nsone.timeout", 5*time.Second, "Timeout for trying to get stats from NSONE.")
	nsoneNumberOfWorkers               = flag.Int("nsone.workers", 50, "Parallel workers that retreives details from NSONE.")
	nsoneNumberOfConcurrentConnections = flag.Int("nsone.number-of-concurrent-connections", 50, "Number of concurrent connections to in parallel to NSONE api.")

	exportUsageByHourFilter = model.NewRegexpOrPanic("off")
	exportUsageByDayFilter = model.NewRegexpOrPanic("off")
	exportUsageByMonthFilter = model.NewRegexpOrPanic(".*")

	exportUsageOfAccount = flag.Bool("export.usage-of-account", true, "Export usages of whole account metric.\n" +
		"\tMetric: 'nsone.usage.account.<period>'")
	exportUsageOfZonesFilter = model.NewRegexpOrPanic(".*")
	exportUsageOfRecordsFilter = model.NewRegexpOrPanic(".*")

	exportQpsOfAccount = flag.Bool("export.qps-of-account", false, "Export queries per second of whole account metric.\n" +
		"\tMetric: 'nsone.qps.account'")
	exportQpsOfZonesFilter = model.NewRegexpOrPanic("off")
	exportQpsOfRecordsFilter = model.NewRegexpOrPanic("off")

	flagsBuffer = &bytes.Buffer{}
)

func main() {
	flag.Var(exportUsageByHourFilter, "export.usage-by-hour-filter", "Export usages by regex of hour metrics.\n" +
		"\tMetric: 'nsone.usage.<dataPoint>.hourly'\n" +
		"\tFor disable: 'off'\n" +
		"\tFor matching account: 'account'\n" +
		"\tFor matching zone: '<zoneName>'\n" +
		"\tFor matching record: '<recordType> <recordName>'")
	flag.Var(exportUsageByDayFilter, "export.usage-by-day-filter", "Export usages by regex of day metrics.\n" +
		"\tMetric: 'nsone.usage.<dataPoint>.daily'\n" +
		"\tFor disable: 'off'\n" +
		"\tFor matching account: 'account'\n" +
		"\tFor matching zone: '<zoneName>'\n" +
		"\tFor matching record: '<recordType> <recordName>'")
	flag.Var(exportUsageByMonthFilter, "export.usage-by-month-filter", "Export usages by regex of month metrics.\n" +
		"\tMetric: 'nsone.usage.<dataPoint>.monthly'\n" +
		"\tFor disable: 'off'\n" +
		"\tFor matching account: 'account'\n" +
		"\tFor matching zone: '<zoneName>'\n" +
		"\tFor matching record: '<recordType> <recordName>'")

	flag.Var(exportUsageOfZonesFilter, "export.usage-of-zones-filter", "Export usages by regex of zone metrics.\n" +
		"\tMetric: 'nsone.usage.zones.<period>'\n" +
		"\tFor disable: 'off'\n" +
		"\tFor matching zone: '<zoneName>'")
	flag.Var(exportUsageOfRecordsFilter, "export.usage-of-records-filter", "Export usages by regex of record metrics.\n" +
		"\tMetric: 'nsone.usage.records.<period>'\n" +
		"\tFor disable: 'off'\n" +
		"\tFor matching record: '<recordType> <recordName>'")

	flag.Var(exportQpsOfZonesFilter, "export.qps-of-zones-filter", "Export queries per second by regex of zone metrics.\n" +
		"\tMetric: 'nsone.qps.zones'\n" +
		"\tFor disable: 'off'\n" +
		"\tFor matching zone: '<zoneName>'")
	flag.Var(exportQpsOfRecordsFilter, "export.qps-of-records-filter", "Export queries per second by regex of record metrics\n" +
		"\tMetric: 'nsone.qps.records'\n" +
		"\tFor disable: 'off'\n" +
		"\tFor matching record: '<recordType> <recordName>'")

	parseUsage()

	exporter := NewNsoneExporter(*nsoneToken, *nsoneTimeout, *nsoneNumberOfWorkers, *nsoneNumberOfConcurrentConnections, NsoneExportSettings{
		UsageByHourFilter:  exportUsageByHourFilter,
		UsageByDayFilter:   exportUsageByDayFilter,
		UsageByMonthFilter: exportUsageByMonthFilter,

		UsageOfAccount: *exportUsageOfAccount,
		UsageOfZonesFilter:   exportUsageOfZonesFilter,
		UsageOfRecordsFilter: exportUsageOfRecordsFilter,

		QpsOfAccount: *exportQpsOfAccount,
		QpsOfZonesFilter:   exportQpsOfZonesFilter,
		QpsOfRecordsFilter: exportQpsOfRecordsFilter,
	})
	prometheus.MustRegister(exporter)

	err := startServer(*metricsPath, *listenAddress, *tlsCert, *tlsPrivateKey, *tlsClientCa)
	if err != nil {
		log.Fatalf("Could not start server. Cause: %v", err)
	}
}

func parseUsage() {
	flags := flag.CommandLine
	flags.SetOutput(flagsBuffer)
	flags.Usage = func() {
		errorString := flagsBuffer.String()
		if len(errorString) > 0 {
			printUsage(strings.TrimSpace(errorString))
		} else {
			printUsage(nil)
		}
	}
	flags.Parse(os.Args[1:])
	assertUsage()
}

func assertUsage() {
	if len(strings.TrimSpace(*listenAddress)) == 0 {
		fail("Missing -web.listen-address")
	}
	if len(strings.TrimSpace(*nsoneToken)) == 0 {
		fail("Missing -nsone.token")
	}
}

func fail(err interface{}) {
	printUsage(err)
	os.Exit(1)
}

func printUsage(err interface{}) {
	fmt.Fprintf(os.Stderr, "%v (version: %v, url: https://github.com/echocat/nsone_exporter)\n", name, version)
	if description != "" {
		fmt.Fprintf(os.Stderr, "%v\n", description)
	}
	fmt.Fprint(os.Stderr, "Author(s): Gregor Noczinski (gregor@noczinski.eu)\n")
	fmt.Fprint(os.Stderr, "\n")

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
	}

	fmt.Fprintf(os.Stderr, "Usage: %v <flags>\n", os.Args[0])
	fmt.Fprint(os.Stderr, "Flags:\n")
	flag.CommandLine.SetOutput(os.Stderr)
	flag.CommandLine.PrintDefaults()
	flag.CommandLine.SetOutput(flagsBuffer)
}
