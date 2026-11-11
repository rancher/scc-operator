ARG GO_IMAGE=registry.suse.com/rancher/appco-go:1.26.11
ARG BCI_VERSION=16.0
ARG FINAL_IMAGE=registry.suse.com/bci/bci-nano:${BCI_VERSION}

FROM registry.suse.com/bci/bci-base:${BCI_VERSION} AS zypper
RUN mkdir -p /chroot/bin /chroot/lib64 /chroot/sbin && \
    cp /bin/bash /chroot/bin/bash && \
    ln -s bash /chroot/bin/sh && \
    cp -a /lib64/. /chroot/lib64/ && \
    cp /sbin/ldconfig /chroot/sbin/ldconfig
RUN zypper -n refresh
RUN zypper -n --installroot /chroot install --no-recommends git grep
RUN rm -f /chroot/bin/bash /chroot/bin/sh
RUN zypper --installroot /chroot clean -a
RUN rm -rf /chroot/var/cache/zypp/* /chroot/var/log/zypp/* /chroot/etc/zypp/ /chroot/run/zypp.pid

FROM ${GO_IMAGE} AS builder
# This doesn't work. Please check:
# https://superuser.com/questions/1828813/docker-error-failed-to-solve-cannot-copy-to-non-directory-when-copying-to
# COPY --from=zypper /chroot /
COPY --from=zypper /chroot/bin /bin
COPY --from=zypper /chroot/lib /lib
COPY --from=zypper /chroot/lib64 /lib64
COPY --from=zypper /chroot/usr /usr

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

FROM registry.suse.com/bci/bci-base:${BCI_VERSION} AS user-setup
RUN useradd -u 1000 -U -m -s /sbin/nologin operator

FROM ${FINAL_IMAGE} AS final
COPY --from=builder /usr/src/app/bin/scc-operator /usr/bin/
COPY --from=user-setup /etc/passwd /etc/passwd
COPY --from=user-setup /etc/group /etc/group
COPY --from=user-setup /etc/shadow /etc/shadow
COPY --from=user-setup --chown=1000:1000 /home/operator /home/operator

USER 1000

ENTRYPOINT ["scc-operator"]
