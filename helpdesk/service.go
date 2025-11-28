package helpdesk

type HelpdeskService struct {
	repo *HelpdeskRepository
}

func NewHelpdeskService(repo *HelpdeskRepository) *HelpdeskService {
	return &HelpdeskService{repo: repo}
}

func (s *HelpdeskService) Create(helpdesk *Helpdesk) error {
	return s.repo.Create(helpdesk)
}

func (s *HelpdeskService) GetAll(limit, offset int, search string) ([]Helpdesk, int, error) {
	helpdesks, err := s.repo.GetAll(limit, offset, search)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.repo.GetTotal(search)
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

func (s *HelpdeskService) UpdateStatus(id int, status string) error {
	return s.repo.UpdateStatus(id, status)
}

func (s *HelpdeskService) Delete(id int) error {
	return s.repo.Delete(id)
}

func (s *HelpdeskService) GetBySessionID(sessionID string) (*Helpdesk, error) {
	return s.repo.GetBySessionID(sessionID)
}