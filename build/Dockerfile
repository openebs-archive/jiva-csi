FROM ubuntu:18.04
RUN apt-get update; exit 0
RUN apt-get -y install rsyslog xfsprogs curl
RUN apt-get clean && rm -rf /var/lib/apt/lists/*

COPY build/bin/jiva-csi /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/jiva-csi"]
