FROM scratch
MAINTAINER contact@echocat.org

COPY build/out/nsone_exporter-linux-amd64 /usr/bin/nsone_exporter

ENTRYPOINT ["/usr/bin/nsone_exporter"]
