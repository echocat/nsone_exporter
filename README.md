# nsone Exporter

nsone exporter for prometheus.io, written in go.

## Get it

### Binary distribution

Download your version from the [releases page](https://github.com/echocat/nsone_exporter/releases/latest). For older version see [archive page](https://github.com/echocat/nsone_exporter/releases).

Example:
```bash
sudo curl -SL https://github.com/echocat/nsone_exporter/releases/download/v0.1.1/nsone_exporter-linux-amd64 \
    > /usr/bin/nsone_exporter
sudo chmod +x /usr/bin/nsone_exporter
```

### Docker image

Image: ``docker pull echocat/nsone_exporter``

You can go to [Docker Hub Tags page](https://hub.docker.com/r/echocat/nsone_exporter/tags/) to see all available tags or you can simply use ``latest``.

## Use it

### Usage

```
Usage: nsone_exporter <flags>
Flags:
  -export.qps-of-account
        Export queries per second of whole account metric.
        Metric: 'nsone.qps.account'
  -export.qps-of-records-filter value
        Export queries per second by regex of record metrics
        Metric: 'nsone.qps.records'
        For disable: 'off'
        For matching record: '<recordType> <recordName>' (default off)
  -export.qps-of-zones-filter value
        Export queries per second by regex of zone metrics.
        Metric: 'nsone.qps.zones'
        For disable: 'off'
        For matching zone: '<zoneName>' (default off)
  -export.usage-by-day-filter value
        Export usages by regex of day metrics.
        Metric: 'nsone.usage.<dataPoint>.daily'
        For disable: 'off'
        For matching account: 'account'
        For matching zone: '<zoneName>'
        For matching record: '<recordType> <recordName>' (default off)
  -export.usage-by-hour-filter value
        Export usages by regex of hour metrics.
        Metric: 'nsone.usage.<dataPoint>.hourly'
        For disable: 'off'
        For matching account: 'account'
        For matching zone: '<zoneName>'
        For matching record: '<recordType> <recordName>' (default off)
  -export.usage-by-month-filter value
        Export usages by regex of month metrics.
        Metric: 'nsone.usage.<dataPoint>.monthly'
        For disable: 'off'
        For matching account: 'account'
        For matching zone: '<zoneName>'
        For matching record: '<recordType> <recordName>' (default .*)
  -export.usage-of-account
        Export usages of whole account metric.
        Metric: 'nsone.usage.account.<period>' (default true)
  -export.usage-of-records-filter value
        Export usages by regex of record metrics.
        Metric: 'nsone.usage.records.<period>'
        For disable: 'off'
        For matching record: '<recordType> <recordName>' (default .*)
  -export.usage-of-zones-filter value
        Export usages by regex of zone metrics.
        Metric: 'nsone.usage.zones.<period>'
        For disable: 'off'
        For matching zone: '<zoneName>' (default .*)
  -log.format value
        If set use a syslog logger or JSON logging. Example: logger:syslog?appname=bob&local=7 or logger:stdout?json=true. Defaults to stderr.
  -log.level value
        Only log messages with the given severity or above. Valid levels: [debug, info, warn, error, fatal]. (default info)
  -nsone.number-of-concurrent-connections int
        Number of concurrent connections to in parallel to NSONE api. (default 50)
  -nsone.timeout duration
        Timeout for trying to get stats from NSONE. (default 5s)
  -nsone.token string
        Token to access the API of nsone.
  -nsone.workers int
        Parallel workers that retreives details from NSONE. (default 50)
  -web.listen-address string
        Address to listen on for web interface and telemetry. (default ":9113")
  -web.telemetry-path string
        Path under which to expose metrics. (default "/metrics")
  -web.tls-cert string
        Path to PEM file that conains the certificate (and optionally also the private key in PEM format).
        This should include the whole certificate chain.
        If provided: The web socket will be a HTTPS socket.
        If not provided: Only HTTP.
  -web.tls-client-ca string
        Path to PEM file that conains the CAs that are trused for client connections.
        If provided: Connecting clients should present a certificate signed by one of this CAs.
        If not provided: Every client will be accepted.
  -web.tls-private-key string
        Path to PEM file that contains the private key (if not contained in web.tls-cert file).
```

### Examples

#### Binary distribution

```bash
# Simply start the exporter with your token and listen on 0.0.0.0:9113
nsone_exporter \
    -nsone.token=mySecrectToken

# Start the exporter with your token and listen on 0.0.0.0:9113
# ...it also secures the connector via SSL 
nsone_exporter \
    -listen.address=:8443 \
    -web.tls-cert=my.server.com.pem

# Simply start the exporter with your token and listen on 0.0.0.0:9113
# ...secures the connector via SSL
# ...and requires client certificates signed by your authority
nsone_exporter \
    -listen.address=:8443 \
    -web.tls-cert=my.server.com.pem \
    -web.tls-client-ca=ca.pem
```

#### Docker image

```bash
# Simply start the exporter with your token and listen on 0.0.0.0:9113
docker run -p9113:9113 echocat/nsone_exporter \
    -nsone.token=mySecrectToken

# Start the exporter with your token and listen on 0.0.0.0:9113
# ...it also secures the connector via SSL 
docker run -p9113:9113 -v/etc/certs:/etc/certs:ro echocat/nsone_exporter \
    -listen.address=:8443 \
    -web.tls-cert=/etc/certs/my.server.com.pem

# Simply start the exporter with your token and listen on 0.0.0.0:9113
# ...secures the connector via SSL
# ...and requires client certificates signed by your authority
docker run -p9113:9113 -v/etc/certs:/etc/certs:ro echocat/nsone_exporter \
    -listen.address=:8443 \
    -web.tls-cert=my.server.com.pem \
    -web.tls-client-ca=ca.pem
```

## Metrics

| Name | Labels | Type | Description |
| ---- | ------ | ---- | ----------- |
| ``nsone_up`` | _none_ | Gauge | Is ``1`` if data could be queried from NSONE. ``0`` if this was not possible |
| ``nsone_qps_account`` | _none_ | Gauge | Queries per second of whole account. |
| ``nsone_qps_zones``   | ``zone`` | Gauge | Queries per second of selected zones. |
| ``nsone_qps_records`` | ``zone``, ``record``, ``recordType`` | Gauge | Queries per second of selected records. |
| ``nsone_usage_account_hourly`` | _none_ | Gauge | Usage of whole account in the last hour. |
| ``nsone_usage_account_daily`` | _none_ | Gauge | Usage of whole account in the last day. |
| ``nsone_usage_account_monthly`` | _none_ | Gauge | Usage of whole account in the last month. |
| ``nsone_usage_zones_hourly`` | ``zone`` | Gauge | Usage of selected zones in the last hour. |
| ``nsone_usage_zones_daily`` | ``zone`` | Gauge | Usage of selected zones in the last day. |
| ``nsone_usage_zones_monthly`` | ``zone`` | Gauge | Usage of selected zones in the last month. |
| ``nsone_usage_records_hourly`` | ``zone``, ``record``, ``recordType`` | Gauge | Usage of selected records in the last hour. |
| ``nsone_usage_records_daily`` | ``zone``, ``record``, ``recordType`` | Gauge | Usage of selected records in the last day. |
| ``nsone_usage_records_monthly`` | ``zone``, ``record``, ``recordType`` | Gauge | Usage of selected records in the last month. |

## Build it

### Precondition

For building nsone_exporter there is only:

1. a compatible operating system (Linux, Windows or Mac OS X)
2. and a working [Java 8](http://www.oracle.com/technetwork/java/javase/downloads/index.html) installation required.

There is no need for a working and installed Go installation (or anything else). The build system will download every dependency and build it if necessary.

> **Hint:** The Go runtime build by the build system will be placed under ``~/.go/sdk``.

### Run build process

On Linux and Mac OS X:
```bash
# Build binaries (includes test)
./gradlew build

# Run tests (but do not build binaries)
./gradlew test

# Build binaries and release it on GitHub
# Environment variable GITHUB_TOKEN is required
./gradlew build githubRelease
```

On Windows:
```bash
# Build binaries (includes test)
gradlew build

# Run tests (but do not build binaries)
gradlew test

# Build binaries and release it on GitHub
# Environment variable GITHUB_TOKEN is required
gradlew build githubRelease
```

### Build artifacts

* Compiled and lined binaries can be found under ``./build/out/nsone_exporter-*``

## Contributing

nsone_exporter is an open source project of [echocat](https://echocat.org).
So if you want to make this project even better, you can contribute to this project on [Github](https://github.com/echocat/nsone_exporter)
by [fork us](https://github.com/echocat/nsone_exporter/fork).

If you commit code to this project you have to accept that this code will be released under the [license](#license) of this project.


## License

See [LICENSE](LICENSE) file.
