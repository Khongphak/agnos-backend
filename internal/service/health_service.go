package service

import (
	"context"
	"time"

	"github.com/agnos-assessment/agnos-backend/internal/repository"
)

const healthPingTimeout = 2 * time.Second

type HealthService interface {
	Check(ctx context.Context) error
}

type healthService struct {
	repo repository.HealthRepository
}

func NewHealthService(repo repository.HealthRepository) HealthService {
	return &healthService{repo: repo}
}

func (s *healthService) Check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, healthPingTimeout)
	defer cancel()
	return s.repo.Ping(ctx)
}
