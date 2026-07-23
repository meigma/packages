FROM debian:13-slim@sha256:020c0d20b9880058cbe785a9db107156c3c75c2ac944a6aa7ab59f2add76a7bd

ARG TOOLS_UID=1000

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        apt-utils \
        ca-certificates \
        createrepo-c \
        curl \
        dpkg-dev \
        file \
        gnupg \
        python3 \
        rpm \
    && rm -rf /var/lib/apt/lists/*

RUN useradd --uid "$TOOLS_UID" --gid nogroup --create-home tools

USER tools

ENTRYPOINT []
