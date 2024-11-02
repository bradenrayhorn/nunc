FROM golang:1.23@sha256:ad5c126b5cf501a8caef751a243bb717ec204ab1aa56dc41dc11be089fafcb4f AS buildgo

RUN mkdir /app
COPY . /app
WORKDIR /app

RUN CGO_ENABLED=0 go build .

FROM alpine@sha256:beefdbd8a1da6d2915566fde36db9db0b524eb737fc57cd1367effd16dc0d06d

RUN mkdir /app
COPY --from=buildgo /app/nunc /app/

COPY --from=buildgo /usr/local/go/LICENSE /app/GO-LICENSE
COPY licenses.txt /app/THIRD-PARTY-LICENSES

CMD ["/app/nunc"]

