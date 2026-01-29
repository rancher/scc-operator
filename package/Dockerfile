FROM registry.suse.com/bci/golang:1.24 AS builder

ARG TAG=''
ARG REPO=''
ENV TAG=$TAG REPO=$REPO

WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN ./scripts/build

FROM registry.suse.com/bci/bci-micro:latest
COPY --from=builder /usr/src/app/bin/scc-operator  /usr/bin/

RUN echo 'scc:x:1000:1000::/home/scc:/bin/bash' >> /etc/passwd && \
    echo 'scc:x:1000:' >> /etc/group && \
    mkdir /home/scc && \
    chown -R scc:scc /home/scc

USER 1000

ENTRYPOINT ["scc-operator"]