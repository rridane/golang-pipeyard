# Étape 1 : base Go
FROM golang:1.23-bookworm

# Installer BuildKit (buildctl et buildkitd)
RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates \
      curl \
      git \
      bash \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /usr/local/bin \
    && curl -sSL https://github.com/moby/buildkit/releases/download/v0.15.1/buildkit-v0.15.1.linux-amd64.tar.gz \
       | tar -xz -C /usr/local/bin --strip-components=1 \
    && chmod +x /usr/local/bin/buildctl /usr/local/bin/buildkitd

# Variables d’environnement minimales
ENV PATH="/go/bin:/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin" \
    BUILDKITD_FLAGS="--oci-worker-no-process-sandbox" \
    DOCKER_BUILDKIT=1

# Par défaut : shell interactif
ENTRYPOINT ["/bin/bash"]

