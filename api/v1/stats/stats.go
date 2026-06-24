package stats

import (
	"net/http"
	"slices"
	"strconv"

	"reverse-watch/domain/dto"
	"reverse-watch/domain/repository"
	"reverse-watch/errors"
	"reverse-watch/middleware"
	"reverse-watch/render"
)

var allowedDays = []int{7, 30, 60, 90, 180, 365}

func summaryHandler(w http.ResponseWriter, r *http.Request) {
	factory := r.Context().Value(middleware.FactoryContextKey).(repository.Factory)

	stats, err := factory.Reversal().SummaryStats()
	if err != nil {
		render.Errorf(w, r, errors.InternalServerError, "failed to load summary stats")
		return
	}

	render.JSON(w, r, stats)
}

type dailyResponse struct {
	Data []dto.DailyCount `json:"data"`
}

func dailyHandler(w http.ResponseWriter, r *http.Request) {
	days := 30
	if daysStr := r.URL.Query().Get("days"); daysStr != "" {
		parsed, err := strconv.Atoi(daysStr)
		if err != nil || !slices.Contains(allowedDays, parsed) {
			render.Errorf(w, r, errors.BadRequest, "days must be one of 7, 30, 60, 90, 180, 365")
			return
		}
		days = parsed
	}

	factory := r.Context().Value(middleware.FactoryContextKey).(repository.Factory)

	counts, err := factory.Reversal().DailyCounts(days)
	if err != nil {
		render.Errorf(w, r, errors.InternalServerError, "failed to load daily counts")
		return
	}

	render.JSON(w, r, dailyResponse{Data: counts})
}
