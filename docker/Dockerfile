# The environment a comparison run happens in.
#
# Randomness is the loudest source of transcript churn but not the largest.
# Interpreter versions, locale, timezone, terminal size, hash seeds and address
# layout all change what a program prints without changing what it does, and a
# difference between two ports means nothing if the two ran in different
# worlds. A seed and a tape make the numbers agree; this makes everything
# around them agree.
#
# Everything is pinned by digest or checksum, so `docker build` produces the
# same toolchains today and next year. The pins are the point -- an image that
# drifts is an image that turns a fixed bug back into an open one.
#
# amd64 only, and deliberately: the reference interpreter is published for
# linux x86_64, and comparing against a reference built for another
# architecture would fold floating-point and library differences into every
# comparison. Building this on arm64 works through emulation and is slow.

ARG BASE_DIGEST=sha256:88200866dfff7ea7f5cbcb6ec7c8a701889efe6fe859fe64d6990e4b07ea4171
FROM --platform=linux/amd64 debian:bookworm-slim@${BASE_DIGEST}
ARG BASE_DIGEST

# Ruby, Python, Perl and Lua come from Debian stable, which is what pinning
# means for a distribution: a fixed series that takes security fixes and no new
# features for the life of the release. Lua's package registers /usr/bin/lua
# through alternatives, which is the name runners/lua.yaml starts.
#
# Go, Node, Java, .NET and the reference interpreter are not in it, so they are
# fetched from upstream and checked against a published checksum before
# anything is unpacked.
ARG GO_VERSION=1.24.13
ARG GO_SHA256=1fc94b57134d51669c72173ad5d49fd62afb0f1db9bf3f798fd98ee423f8d730
ARG NODE_VERSION=24.20.0
ARG NODE_SHA256=2f2c0da162318f0de47665410c7c8c2ed3d36c8f3105de4bbc61176c70a7cbf2
# Temurin rather than Debian's default-jdk, which is 17: the Java shim rewrites
# the port's call sites with java.lang.classfile, standard since 24. 25 is the
# long-term release above that floor. The tag carries a + that a URL cannot.
ARG JAVA_VERSION=25.0.4.1_1
ARG JAVA_TAG=jdk-25.0.4.1%2B1
ARG JAVA_SHA256=dbb698396d478e7fa2b1e50f4103324b2a99b90569ee27c33f2261f9215cf41e
# The SDK rather than the runtime: a .NET port arrives as source, and `dotnet
# run` compiles it. Checked against the sha512 Microsoft publishes beside the
# tarball, which is the checksum there is to check it against.
ARG DOTNET_VERSION=10.0.400
ARG DOTNET_SHA512=1033977dd837150e0814cf0c5d5b17ceb63925fda7ba2158b47258a4bd7c048cf82eac3bc1166f3146f53124a3f5fba09db1de1260d2ce96399860303b404b48
ARG VINTBAS_RELEASE=vintbas-1.0.3-1
ARG VINTBAS_SHA256=72dda1d16ed21f0d812f6978a32d105dadea49dd8163692b44c462678eb16a17

SHELL ["/bin/bash", "-o", "pipefail", "-euc"]

RUN apt-get update \
 && apt-get install --no-install-recommends --yes \
      ca-certificates \
      curl \
      git \
      lua5.4 \
      make \
      perl \
      python3 \
      ruby \
      tzdata \
      util-linux \
      xz-utils \
 && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL -o /tmp/go.tar.gz "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" \
 && echo "${GO_SHA256}  /tmp/go.tar.gz" | sha256sum --check --strict - \
 && tar --directory /usr/local --extract --file /tmp/go.tar.gz \
 && rm /tmp/go.tar.gz

RUN curl -fsSL -o /tmp/node.tar.xz "https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-x64.tar.xz" \
 && echo "${NODE_SHA256}  /tmp/node.tar.xz" | sha256sum --check --strict - \
 && tar --directory /usr/local --extract --file /tmp/node.tar.xz --strip-components 1 \
      --exclude CHANGELOG.md --exclude LICENSE --exclude README.md \
 && rm /tmp/node.tar.xz

RUN curl -fsSL -o /tmp/jdk.tar.gz \
      "https://github.com/adoptium/temurin25-binaries/releases/download/${JAVA_TAG}/OpenJDK25U-jdk_x64_linux_hotspot_${JAVA_VERSION}.tar.gz" \
 && echo "${JAVA_SHA256}  /tmp/jdk.tar.gz" | sha256sum --check --strict - \
 && mkdir -p /usr/local/jdk \
 && tar --directory /usr/local/jdk --extract --file /tmp/jdk.tar.gz --strip-components 1 \
 && rm /tmp/jdk.tar.gz

# libicu is what the .NET runtime uses for anything culture-aware, and without
# it the host refuses to start rather than quietly falling back -- which is the
# right way round, since a program that formats a number differently depending
# on which library it found is exactly the churn this image exists to remove.
RUN apt-get update \
 && apt-get install --no-install-recommends --yes libicu72 \
 && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL -o /tmp/dotnet.tar.gz \
      "https://builds.dotnet.microsoft.com/dotnet/Sdk/${DOTNET_VERSION}/dotnet-sdk-${DOTNET_VERSION}-linux-x64.tar.gz" \
 && echo "${DOTNET_SHA512}  /tmp/dotnet.tar.gz" | sha512sum --check --strict - \
 && mkdir -p /usr/local/dotnet \
 && tar --directory /usr/local/dotnet --extract --file /tmp/dotnet.tar.gz \
 && rm /tmp/dotnet.tar.gz

# The reference BASIC interpreter, patched to seed its RNG from VINTBAS_SEED.
# The same binary the Makefile installs, so a transcript recorded here and one
# recorded on a developer's machine came from the same interpreter.
RUN curl -fsSL -o /usr/local/bin/vintbas \
      "https://github.com/rjnienaber/prescript/releases/download/${VINTBAS_RELEASE}/vintbas-linux-x86_64" \
 && echo "${VINTBAS_SHA256}  /usr/local/bin/vintbas" | sha256sum --check --strict - \
 && chmod +x /usr/local/bin/vintbas

ENV PATH=/usr/local/go/bin:/usr/local/jdk/bin:/usr/local/dotnet:/usr/local/bin:$PATH
ENV DOTNET_ROOT=/usr/local/dotnet

# The rest of what moves. Each of these has been a transcript diff at some
# point in somebody's life:
#
#   LANG/LC_ALL  decimal separators, sort order, and how a byte over 127 prints
#   TZ           any program that prints a date
#   COLUMNS      a program that wraps its own output; belt and braces with the
#                fixed pty size prescript gives the child
#   PYTHONHASHSEED  the iteration order of a set or a dict of strings
#   GOTOOLCHAIN  stops the go command downloading a different Go than the one
#                pinned above, which would quietly unpin the image
#   DOTNET_*     the first `dotnet` command on a fresh machine prints a welcome
#                banner and sends a telemetry ping, either of which would
#                arrive in the middle of a transcript
ENV LANG=C.UTF-8 \
    LC_ALL=C.UTF-8 \
    TZ=UTC \
    COLUMNS=80 \
    LINES=24 \
    PYTHONHASHSEED=0 \
    PYTHONDONTWRITEBYTECODE=1 \
    GOTOOLCHAIN=local \
    DOTNET_NOLOGO=1 \
    DOTNET_CLI_TELEMETRY_OPTOUT=1

# What a bug report has to carry, written from the binaries that were actually
# installed rather than from the arguments above, so it cannot claim a version
# the image does not have. `prescript play --bug-report` quotes it verbatim; a
# human can cat it.
ENV PRESCRIPT_ENVIRONMENT=/etc/prescript-environment
RUN { \
      echo "base=debian:bookworm-slim@${BASE_DIGEST}"; \
      echo "arch=$(uname -m)"; \
      echo "go=$(go version | cut -d' ' -f3)"; \
      echo "node=$(node --version)"; \
      echo "java=$(java --version | head -n 1 | cut -d' ' -f2)"; \
      echo "dotnet=$(dotnet --version)"; \
      echo "python=$(python3 --version | cut -d' ' -f2)"; \
      echo "ruby=$(ruby --version | cut -d' ' -f2)"; \
      echo "perl=$(perl -e 'print $^V')"; \
      echo "lua=$(lua -v | cut -d' ' -f2)"; \
      echo "vintbas=${VINTBAS_RELEASE}"; \
      echo "locale=${LC_ALL}"; \
      echo "timezone=${TZ}"; \
    } > "${PRESCRIPT_ENVIRONMENT}"

COPY entrypoint.sh /usr/local/bin/prescript-entrypoint
RUN chmod +x /usr/local/bin/prescript-entrypoint

# The repository is mounted from the host, so its files belong to a user this
# container does not have, and git refuses to read a repository it considers
# somebody else's. `go build` asks git for the revision it stamps into the
# binary and treats that refusal as an error rather than as "no VCS here", so
# without this every build inside the image fails. Scoped to the one directory
# that is ever mounted.
RUN git config --system --add safe.directory /workspace

WORKDIR /workspace
ENTRYPOINT ["/usr/local/bin/prescript-entrypoint"]
CMD ["bash"]
