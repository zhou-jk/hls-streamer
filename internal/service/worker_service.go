package service

import (
	"time"

	"github.com/Zhou-JK/hls-streamer/internal/model"
	"github.com/Zhou-JK/hls-streamer/internal/repository"
)

type WorkerService struct {
	workerRepo *repository.WorkerRepo
}

func NewWorkerService(workerRepo *repository.WorkerRepo) *WorkerService {
	return &WorkerService{workerRepo: workerRepo}
}

type RegisterWorkerInput struct {
	ID           string     `json:"id" binding:"required"`
	Hostname     string     `json:"hostname" binding:"required"`
	IPAddress    string     `json:"ip_address" binding:"required"`
	Capabilities model.JSON `json:"capabilities"`
}

func (s *WorkerService) Register(input RegisterWorkerInput) error {
	w := &model.Worker{
		ID:            input.ID,
		Hostname:      input.Hostname,
		IPAddress:     input.IPAddress,
		Capabilities:  input.Capabilities,
		Status:        "online",
		LastHeartbeat: time.Now(),
	}
	return s.workerRepo.Upsert(w)
}

func (s *WorkerService) Heartbeat(workerID string) error {
	return s.workerRepo.UpdateHeartbeat(workerID)
}

func (s *WorkerService) SetBusy(workerID string, taskID *uint) error {
	if err := s.workerRepo.UpdateStatus(workerID, "busy"); err != nil {
		return err
	}
	return s.workerRepo.SetCurrentTask(workerID, taskID)
}

func (s *WorkerService) SetOnline(workerID string) error {
	if err := s.workerRepo.UpdateStatus(workerID, "online"); err != nil {
		return err
	}
	return s.workerRepo.SetCurrentTask(workerID, nil)
}

func (s *WorkerService) List() ([]model.Worker, error) {
	return s.workerRepo.List()
}
