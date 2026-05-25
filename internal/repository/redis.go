package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/Endea4/studExE4-location-service/internal/model"
	"github.com/user/studexe4/shared/events"
	"github.com/redis/go-redis/v9"
)

const geoKey = "entities:locations"

type LocationRepository struct {
	rdb *redis.Client
}

func NewLocationRepository(rdb *redis.Client) *LocationRepository {
	return &LocationRepository{rdb: rdb}
}

func (r *LocationRepository) UpdateLocation(ctx context.Context, loc *model.LocationUpdate) error {
	pipe := r.rdb.Pipeline()
	pipe.GeoAdd(ctx, geoKey, &redis.GeoLocation{
		Name:      loc.RefID,
		Longitude: loc.Longitude,
		Latitude:  loc.Latitude,
	})
	pipe.Set(ctx, fmt.Sprintf("entity:loc:%s", loc.RefID), time.Now().UnixMilli(), 0)

	evtData, _ := json.Marshal(events.LocationUpdatedData{
		RefID:     loc.RefID,
		Latitude:  loc.Latitude,
		Longitude: loc.Longitude,
		Timestamp: time.Now().UnixMilli(),
	})
	pipe.Publish(ctx, events.RedisChannelLocationUpdates, evtData)

	_, err := pipe.Exec(ctx)
	return err
}

func (r *LocationRepository) GetLocation(ctx context.Context, refID string) (*model.TrackedEntity, error) {
	results, err := r.rdb.GeoPos(ctx, geoKey, refID).Result()
	if err != nil {
		return nil, err
	}
	if len(results) == 0 || results[0] == nil {
		return nil, nil
	}

	tsStr, err := r.rdb.Get(ctx, fmt.Sprintf("entity:loc:%s", refID)).Result()
	var updatedAt time.Time
	if err == nil {
		ms, _ := strconv.ParseInt(tsStr, 10, 64)
		updatedAt = time.UnixMilli(ms)
	} else {
		updatedAt = time.Now()
	}

	return &model.TrackedEntity{
		RefID:     refID,
		Latitude:  results[0].Latitude,
		Longitude: results[0].Longitude,
		UpdatedAt: updatedAt,
	}, nil
}

func (r *LocationRepository) FindNearby(ctx context.Context, query *model.NearbyQuery) ([]model.NearbyEntity, error) {
	results, err := r.rdb.GeoSearchLocation(ctx, geoKey, &redis.GeoSearchLocationQuery{
		GeoSearchQuery: redis.GeoSearchQuery{
			Longitude:  query.Longitude,
			Latitude:   query.Latitude,
			Radius:     query.RadiusKm,
			RadiusUnit: "km",
			Sort:       "ASC",
		},
		WithCoord: true,
		WithDist:  true,
	}).Result()
	if err != nil {
		return nil, err
	}

	entities := make([]model.NearbyEntity, 0, len(results))
	for _, loc := range results {
		entities = append(entities, model.NearbyEntity{
			RefID:     loc.Name,
			Latitude:  loc.Latitude,
			Longitude: loc.Longitude,
			Distance:  loc.Dist,
		})
	}
	return entities, nil
}

func (r *LocationRepository) GetAllLocations(ctx context.Context) ([]model.TrackedEntity, error) {
	vals, err := r.rdb.ZRange(ctx, geoKey, 0, -1).Result()
	if err != nil {
		return nil, err
	}
	if len(vals) == 0 {
		return nil, nil
	}

	positions, err := r.rdb.GeoPos(ctx, geoKey, vals...).Result()
	if err != nil {
		return nil, err
	}

	locations := make([]model.TrackedEntity, 0, len(vals))
	for i, name := range vals {
		if i < len(positions) && positions[i] != nil {
			tsStr, _ := r.rdb.Get(ctx, fmt.Sprintf("entity:loc:%s", name)).Result()
			var updatedAt time.Time
			if ms, err := strconv.ParseInt(tsStr, 10, 64); err == nil {
				updatedAt = time.UnixMilli(ms)
			} else {
				updatedAt = time.Now()
			}
			locations = append(locations, model.TrackedEntity{
				RefID:     name,
				Latitude:  positions[i].Latitude,
				Longitude: positions[i].Longitude,
				UpdatedAt: updatedAt,
			})
		}
	}
	return locations, nil
}

func (r *LocationRepository) RemoveEntity(ctx context.Context, refID string) error {
	pipe := r.rdb.Pipeline()
	pipe.ZRem(ctx, geoKey, refID)
	pipe.Del(ctx, fmt.Sprintf("entity:loc:%s", refID))
	_, err := pipe.Exec(ctx)
	return err
}
