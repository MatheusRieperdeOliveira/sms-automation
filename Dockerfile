FROM golang:1.25-alpine

RUN apk add --no-cache gcc musl-dev

WORKDIR /sms-automation

COPY . .

CMD ["tail", "-f", "/dev/null"]