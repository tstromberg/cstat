

GO = go

all: cstat cstat-to-csv mstat

cstat: ./cmd/cstat/cstat.go
	$(GO) build -o $@ $^

mstat: ./cmd/mstat/mstat.go
	$(GO) build -o $@ $^

cstat-to-csv: ./cmd/cstat-to-csv/cstat-to-csv.go
	$(GO) build -o $@ $^

clean:
	$(RM) cstat cstat-to-csv
