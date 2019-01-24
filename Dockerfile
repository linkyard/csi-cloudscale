# Copyright 2018 DigitalOcean
# Copyright 2019 cloudscale.ch
# Copyright 2019 linkyard
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang:alpine AS builder

RUN mkdir -p /go/src/github.com/cloudscale-ch/csi-cloudscale \
  && apk add --no-cache bash build-base git

COPY .git /go/src/github.com/cloudscale-ch/csi-cloudscale/.git
COPY cmd /go/src/github.com/cloudscale-ch/csi-cloudscale/cmd
COPY driver /go/src/github.com/cloudscale-ch/csi-cloudscale/driver
COPY vendor /go/src/github.com/cloudscale-ch/csi-cloudscale/vendor
COPY Makefile /go/src/github.com/cloudscale-ch/csi-cloudscale/
COPY util /go/src/github.com/cloudscale-ch/csi-cloudscale/util

WORKDIR /go/src/github.com/cloudscale-ch/csi-cloudscale

RUN make test && make compile


FROM alpine:3.7

RUN apk add --no-cache ca-certificates e2fsprogs findmnt cryptsetup

COPY --from=builder /go/src/github.com/cloudscale-ch/csi-cloudscale/cmd/cloudscale-csi-plugin/cloudscale-csi-plugin /bin/
COPY util/csi-diskinfo.sh /bin/csi-diskinfo.sh

ENTRYPOINT ["/bin/cloudscale-csi-plugin"]
