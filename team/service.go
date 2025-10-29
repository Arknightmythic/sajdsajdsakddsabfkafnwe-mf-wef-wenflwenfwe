package team

type TeamService struct {
	repo *TeamRepository
}

func NewTeamService(repo *TeamRepository) *TeamService {
	return &TeamService{repo: repo}
}

func (s *TeamService) Create(team *Team) error {
	return s.repo.Create(team)
}

func (s *TeamService) GetAll(limit, offset int) ([]Team, int, error) {
	teams, err := s.repo.GetAll(limit, offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.repo.GetTotal()
	if err != nil {
		return nil, 0, err
	}

	return teams, total, nil
}

func (s *TeamService) GetByID(id int) (*Team, error) {
	return s.repo.GetByID(id)
}

func (s *TeamService) Update(team *Team) error {
	return s.repo.Update(team)
}

func (s *TeamService) Delete(id int) error {
	return s.repo.Delete(id)
}