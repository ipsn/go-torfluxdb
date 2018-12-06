# Use the pre-built torfluxproxy image to dedup build tasks
FROM ipsn/torfluxproxy as proxy

# Pull the Tor proxy into a second stage deploy InfluxDB container
FROM influxdb:alpine

COPY --from=proxy /usr/local/bin/torfluxproxy /usr/local/bin/

RUN \
  echo 'torfluxproxy --printkey &'  > torfluxdb.sh && \
  echo '/init-influxdb.sh'         >> torfluxdb.sh && \
  echo 'exec influxd'              >> torfluxdb.sh

ENTRYPOINT ["/bin/sh", "torfluxdb.sh"]
