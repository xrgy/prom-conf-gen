FROM        quay.io/prometheus/busybox:latest
USER root

COPY prom-conf-gen                         /bin/prom-conf-gen
COPY /template    /ruletemplates




EXPOSE     8090
ENTRYPOINT ["/bin/prom-conf-gen"]
