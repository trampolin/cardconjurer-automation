package api

import (
	"cardconjurer-automation/pkg/cardconjurer"
	"go.uber.org/zap"
	"net/http"
)

type Api struct {
	logger *zap.SugaredLogger
	cc     *cardconjurer.CardConjurer
}

func New(logger *zap.SugaredLogger, cc *cardconjurer.CardConjurer) *Api {
	return &Api{
		logger: logger,
		cc:     cc,
	}
}

func (a *Api) HelloHandler(w http.ResponseWriter, _ *http.Request) {
	a.logger.Info("HelloHandler called")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("Hello, CardConjurer!"))
	if err != nil {
		a.logger.Errorw("Error writing response", "error", err)
	}
}
