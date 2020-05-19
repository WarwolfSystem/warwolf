# The Warwolf System
# Copyright (C) 2020 The Warwolf Authors

# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as
# published by the Free Software Foundation, either version 3 of the
# License, or (at your option) any later version.

# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU Affero General Public License for more details.

# You should have received a copy of the GNU Affero General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.

FROM golang:alpine as builder
COPY . /tmp/wwf
RUN cd /tmp/wwf && CGO_ENABLED=0 go build -ldflags "-w -s"
FROM alpine:latest
ENV WWFAs=Client \
    WWFBackend= \
    WWFKey=TheRightToCommunicateFreelyPrivatelySecretlyAndSecurelyIsEssentialForASafeSociety \
    WWFListen=0.0.0.0:1080 \
    WWFBackendHostEnforce= \
    WWFUsername= \
    WWFPassword= \
    WWFMaxClientConnections=128 \
    WWFMaxBackendConnections=4 \
    WWFMaxRetrieveLength=64946 \
    WWFRequestTimeout=32 \
    WWFIdleTimeout=128 \
    WWFMaxRetries=16
COPY --from=builder /tmp/wwf/warwolf /bin/wwf
COPY . /src
RUN apk add --no-cache ca-certificates && update-ca-certificates && echo "net.ipv4.ip_local_port_range = 1081 2000" >> /etc/sysctl.conf && addgroup -S wwf && adduser -S -G wwf wwf
USER wwf
EXPOSE 1080/tcp 1081-2000/udp
ENTRYPOINT [ "/bin/wwf" ]
