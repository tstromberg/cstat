

GO = go

all: cstat cstat-to-csv dstat

cstat: ./cmd/cstat/cstat.go
	$(GO) build -o $@ $^

dstat: ./cmd/dstat/dstat.go
	$(GO) build -o $@ $^

cstat-to-csv: ./cmd/cstat-to-csv/cstat-to-csv.go
	$(GO) build -o $@ $^

clean:
	$(RM) cstat cstat-to-csv
