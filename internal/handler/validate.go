package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	api "github.com/StasanDev/avitoLabs/internal/generated"
	"github.com/google/uuid"
)

func validateRequiredTripFields(payload []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		return fmt.Errorf("decode request fields: %w", err)
	}

	for _, field := range []string{"user_id", "driver_id", "start_point", "end_point", "price"} {
		if err := requireJSONField(fields, field); err != nil {
			return err
		}
	}

	for _, point := range []string{"start_point", "end_point"} {
		var coordinates map[string]json.RawMessage
		if err := json.Unmarshal(fields[point], &coordinates); err != nil {
			return fmt.Errorf("%s must be an object", point)
		}
		for _, coordinate := range []string{"latitude", "longitude"} {
			if err := requireJSONField(coordinates, coordinate); err != nil {
				return fmt.Errorf("%s.%w", point, err)
			}
		}
	}

	return nil
}

func requireJSONField(fields map[string]json.RawMessage, name string) error {
	value, ok := fields[name]
	if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		return fmt.Errorf("field %s is required", name)
	}
	return nil
}

func validateTripData(body api.TripData) error {
	if body.UserId == uuid.Nil {
		return errors.New("user_id must be a non-empty UUID")
	}
	if body.DriverId == uuid.Nil {
		return errors.New("driver_id must be a non-empty UUID")
	}
	if err := validateCoordinates("start_point", body.StartPoint); err != nil {
		return err
	}
	if err := validateCoordinates("end_point", body.EndPoint); err != nil {
		return err
	}
	if body.Price < 0 {
		return errors.New("price must be greater than or equal to zero")
	}
	return nil
}

func validateCoordinates(name string, coordinates api.Coordinates) error {
	if coordinates.Latitude < -90 || coordinates.Latitude > 90 {
		return fmt.Errorf("%s.latitude must be between -90 and 90", name)
	}
	if coordinates.Longitude < -180 || coordinates.Longitude > 180 {
		return fmt.Errorf("%s.longitude must be between -180 and 180", name)
	}
	return nil
}
