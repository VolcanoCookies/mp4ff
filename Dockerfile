FROM golang:latest

WORKDIR /app

RUN go install github.com/Eyevinn/mp4ff/cmd/mp4ff-info@latest
RUN curl https://cdn.discordapp.com/attachments/1056944061732368415/1176825637877600266/test.mp4 -o test.mp4

CMD ["mp4ff-info", "test.mp4"]