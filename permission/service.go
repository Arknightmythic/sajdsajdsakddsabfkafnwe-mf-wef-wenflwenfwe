package permission

type PermissionService struct {
	repo *PermissionRepository
}

func NewPermissionService(repo *PermissionRepository) *PermissionService {
	return &PermissionService{repo: repo}
}

func (s *PermissionService) Create(permission *Permission) error {
	return s.repo.Create(permission)
}

func (s *PermissionService) GetAll() ([]Permission, error) {
	return s.repo.GetAll()
}

func (s *PermissionService) GetByID(id int) (*Permission, error) {
	return s.repo.GetByID(id)
}

func (s *PermissionService) Update(permission *Permission) error {
	return s.repo.Update(permission)
}

func (s *PermissionService) Delete(id int) error {
	return s.repo.Delete(id)
}
