package usecase

type TripService struct {
	tripRepository tripRepository
}

func NewTripService(tripRepository tripRepository) *TripService {
	return &TripService{
		tripRepository: tripRepository,
	}
}
