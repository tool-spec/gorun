package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/hydrocode-de/gorun/internal/cache"
	"github.com/hydrocode-de/gorun/internal/db"
	"github.com/hydrocode-de/gorun/internal/service"
	"github.com/spf13/viper"
)

func getService() *service.Service {
	return &service.Service{
		DB:    viper.Get("db").(*db.Queries),
		Cache: viper.Get("cache").(*cache.Cache),
	}
}

func runFromRequest(ctx context.Context, r *http.Request, DB *db.Queries, userID string) (db.Run, error) {
	idPath := r.PathValue("id")
	if idPath == "" {
		return db.Run{}, fmt.Errorf("missing run id")
	}
	id, err := strconv.ParseInt(idPath, 10, 64)
	if err != nil {
		return db.Run{}, fmt.Errorf("the passed run id is not a valid integer: %w", err)
	}
	return DB.GetRun(ctx, db.GetRunParams{ID: id, UserID: userID})
}
