package service

import "github.com/agnos-assessment/agnos-backend/internal/repository"

type HealthService interface {
	Check() (status string, database string)
}

type healthService struct {
	repo repository.HealthRepository
}

func NewHealthService(repo repository.HealthRepository) HealthService {
	return &healthService{repo: repo}
}

func (s *healthService) Check() (string, string) {
	if err := s.repo.Ping(); err != nil {
		return "ok", "disconnected"
	}
	return "ok", "connected"
}
