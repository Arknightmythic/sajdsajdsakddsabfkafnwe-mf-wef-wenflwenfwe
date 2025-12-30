package helpdesk

import (
	"time"

	"github.com/google/uuid"
)

type HelpdeskService struct {
	repo *HelpdeskRepository
}

func NewHelpdeskService(repo *HelpdeskRepository) *HelpdeskService {
	return &HelpdeskService{repo: repo}
}

func (s *HelpdeskService) GetSwitchStatus() (*SwitchHelpdesk, error) {
	return s.repo.GetSwitchStatus()
}

func (s *HelpdeskService) UpdateSwitchStatus(status bool) (*SwitchHelpdesk, error) {
	return s.repo.UpdateSwitchStatus(status)
}

func (s *HelpdeskService) Create(helpdesk *Helpdesk) error {
	return s.repo.Create(helpdesk)
}


func (s *HelpdeskService) GetAll(limit, offset int, search string, status string) ([]Helpdesk, int, error) {

	helpdesks, total, err := s.repo.GetAll(limit, offset, search, status)
	if err != nil {
		return nil, 0, err
	}
	return helpdesks, total, nil
}

func (s *HelpdeskService) GetByID(id int) (*Helpdesk, error) {
	return s.repo.GetByID(id)
}

func (s *HelpdeskService) Update(helpdesk *Helpdesk) error {
	return s.repo.Update(helpdesk)
}

func (s *HelpdeskService) UpdateStatus(id int, status string, userID any) error {
	return s.repo.UpdateStatus(id, status, userID)
}

func (s *HelpdeskService) Delete(id int) error {
	return s.repo.Delete(id)
}

func (s *HelpdeskService) GetBySessionID(sessionID string) (*Helpdesk, error) {
	return s.repo.GetBySessionID(sessionID)
}

func (s *HelpdeskService) SolvedConversation(id uuid.UUID) error {
	const customLayout = "2006-01-02 15:04:05.000"
	now := time.Now()
	formattedTime := now.Format(customLayout)

	err := s.repo.SolvedConversation(id)
	if err != nil {
		return err
	}

	err = s.repo.EndTimestampConversation(id, formattedTime)

	if err != nil {
		return err
	}

	return nil
}

func (s *HelpdeskService) GetSummary() (*HelpdeskSummary, error) {
	return s.repo.GetSummary()
}
