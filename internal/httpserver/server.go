package httpserver

import (
	"github.com/Osselnet/metrics-collector/internal/handlers"
	"github.com/Osselnet/metrics-collector/internal/storage"
	"net/http"
)

type Config struct {
	Address string
	Port    string
}

func New(cfg Config) {
	st := storage.New()
	gaugeHandler := &handlers.Gauge{MemStorage: st}
	counterHandler := &handlers.Counter{MemStorage: st}

	//http://<АДРЕС_СЕРВЕРА>/update/<ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>/<ЗНАЧЕНИЕ_МЕТРИКИ>
	mux := http.NewServeMux()
	mux.Handle("/update/gauge/", gaugeHandler)
	mux.Handle("/update/counter/", counterHandler)

	mainHandler := http.Handler(mux)
	mainHandler = preChecksMiddleware(mainHandler)
	mainHandler = accessLogMiddleware(mainHandler)
	mainHandler = panicMiddleware(mainHandler)

	addr := cfg.Address + ":" + cfg.Port
	err := http.ListenAndServe(addr, mainHandler)
	if err != nil {
		panic(err)
	}
}
