FROM ubuntu:16.04
RUN apt-get update; exit 0
RUN apt-get -y install rsyslog xfsprogs
RUN apt-get clean && rm -rf /var/lib/apt/lists/*

COPY build/bin/jiva-csi /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/jiva-csi"]
