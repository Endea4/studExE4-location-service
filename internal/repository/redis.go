package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Endea4/studExE4-location-service/internal/model"
	"github.com/redis/go-redis/v9"
)

const geoKey = "drivers:locations"

type LocationRepository struct {
	rdb *redis.Client
}

func NewLocationRepository(rdb *redis.Client) *LocationRepository {
	return &LocationRepository{rdb: rdb}
}

func (r *LocationRepository) UpdateLocation(ctx context.Context, loc *model.LocationUpdate) error {
	pipe := r.rdb.Pipeline()
	pipe.GeoAdd(ctx, geoKey, &redis.GeoLocation{
		Name:      loc.DriverID,
		Longitude: loc.Longitude,
		Latitude:  loc.Latitude,
	})
	pipe.Set(ctx, fmt.Sprintf("driver:loc:%s", loc.DriverID), time.Now().UnixMilli(), 0)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *LocationRepository) GetLocation(ctx context.Context, driverID string) (*model.DriverLocation, error) {
	results, err := r.rdb.GeoPos(ctx, geoKey, driverID).Result()
	if err != nil {
		return nil, err
	}
	if len(results) == 0 || results[0] == nil {
		return nil, nil
	}

	tsStr, err := r.rdb.Get(ctx, fmt.Sprintf("driver:loc:%s", driverID)).Result()
	var updatedAt time.Time
	if err == nil {
		ms, _ := strconv.ParseInt(tsStr, 10, 64)
		updatedAt = time.UnixMilli(ms)
	} else {
		updatedAt = time.Now()
	}

	return &model.DriverLocation{
		DriverID:  driverID,
		Latitude:  results[0].Latitude,
		Longitude: results[0].Longitude,
		UpdatedAt: updatedAt,
	}, nil
}

func (r *LocationRepository) FindNearby(ctx context.Context, query *model.NearbyQuery) ([]model.NearbyDriver, error) {
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

	drivers := make([]model.NearbyDriver, 0, len(results))
	for _, loc := range results {
		drivers = append(drivers, model.NearbyDriver{
			DriverID:  loc.Name,
			Latitude:  loc.Latitude,
			Longitude: loc.Longitude,
			Distance:  loc.Dist,
		})
	}
	return drivers, nil
}

func (r *LocationRepository) GetAllLocations(ctx context.Context) ([]model.DriverLocation, error) {
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

	locations := make([]model.DriverLocation, 0, len(vals))
	for i, name := range vals {
		if i < len(positions) && positions[i] != nil {
			tsStr, _ := r.rdb.Get(ctx, fmt.Sprintf("driver:loc:%s", name)).Result()
			var updatedAt time.Time
			if ms, err := strconv.ParseInt(tsStr, 10, 64); err == nil {
				updatedAt = time.UnixMilli(ms)
			} else {
				updatedAt = time.Now()
			}
			locations = append(locations, model.DriverLocation{
				DriverID:  name,
				Latitude:  positions[i].Latitude,
				Longitude: positions[i].Longitude,
				UpdatedAt: updatedAt,
			})
		}
	}
	return locations, nil
}

func (r *LocationRepository) RemoveDriver(ctx context.Context, driverID string) error {
	pipe := r.rdb.Pipeline()
	pipe.ZRem(ctx, geoKey, driverID)
	pipe.Del(ctx, fmt.Sprintf("driver:loc:%s", driverID))
	_, err := pipe.Exec(ctx)
	return err
}
