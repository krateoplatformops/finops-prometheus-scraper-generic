FROM scratch
WORKDIR /prometheus-scraper-generic
COPY ./bin ./bin
WORKDIR /temp
WORKDIR /prometheus-scraper-generic/bin
ENTRYPOINT ["./prometheus-scraper-generic"]