-- +goose Up
CREATE UNIQUE INDEX trips_driver_active_uidx ON trips (driver_id) WHERE status = 'active';

-- +goose Down
DROP INDEX trips_driver_active_uidx;
