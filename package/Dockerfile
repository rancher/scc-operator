ARG GO_IMAGE=rancher/hardened-build-base:v1.26.8b1
FROM ${GO_IMAGE} AS builder

ARG TAG=''
ARG REPO=''
ENV TAG=$TAG REPO=$REPO

WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    ./scripts/build

FROM registry.suse.com/bci/bci-base:latest AS user-setup
RUN useradd -u 1000 -U -m -s /sbin/nologin operator

FROM registry.suse.com/bci/bci-nano:latest
COPY --from=builder /usr/src/app/bin/scc-operator  /usr/bin/
COPY --from=user-setup /etc/passwd /etc/passwd
COPY --from=user-setup /etc/group /etc/group
COPY --from=user-setup /etc/shadow /etc/shadow
COPY --from=user-setup --chown=1000:1000 /home/operator /home/operator

USER 1000

ENTRYPOINT ["scc-operator"]
