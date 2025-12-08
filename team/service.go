package team


import (
	"dokuprime-be/permission" 
	"strconv"
	"strings"
)

type TeamService struct {
	repo *TeamRepository
	repoPermission *permission.PermissionRepository
}

func NewTeamService(repo *TeamRepository, repoPermission *permission.PermissionRepository) *TeamService {
	return &TeamService{
		repo:           repo,
		repoPermission: repoPermission,
	}
}

func (s *TeamService) Create(team *Team) error {
	return s.repo.Create(team)
}

func (s *TeamService) GetAll(limit, offset int, search string) ([]Team, int, error) {
	teams, err := s.repo.GetAll(limit, offset, search)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.repo.GetTotal(search)
	if err != nil {
		return nil, 0, err
	}

	return teams, total, nil
}

func (s *TeamService) GetByID(id int) (*Team, error) {
	return s.repo.GetByID(id)
}

func (s *TeamService) Update(team *Team) error {
	
	oldTeam, err := s.repo.GetByID(team.ID)
	if err != nil {
		return err
	}

	
	removedPages := s.getRemovedPages(oldTeam.Pages, team.Pages)

	
	if len(removedPages) > 0 {
		if err := s.processPermissionRevocation(team.ID, removedPages); err != nil {
			return err
		}
	}

	
	return s.repo.Update(team)
}





func (s *TeamService) getRemovedPages(oldPages, newPages []string) []string {
	newPagesMap := make(map[string]bool)
	for _, p := range newPages {
		newPagesMap[p] = true
	}

	var removedPages []string
	for _, p := range oldPages {
		if !newPagesMap[p] {
			removedPages = append(removedPages, p)
		}
	}
	return removedPages
}

func (s *TeamService) processPermissionRevocation(teamID int, removedPages []string) error {
	allPerms, err := s.repoPermission.GetAll()
	if err != nil {
		return err
	}

	bannedPermIDs := s.findBannedPermissionIDs(allPerms, removedPages)

	if len(bannedPermIDs) > 0 {
		return s.repo.RevokeRolePermissions(teamID, bannedPermIDs)
	}
	return nil
}

func (s *TeamService) findBannedPermissionIDs(allPerms []permission.Permission, removedPages []string) []string {
	var bannedPermIDs []string
	for _, perm := range allPerms {
		for _, page := range removedPages {
			
			prefix := page + ":"
			if strings.HasPrefix(perm.Name, prefix) {
				bannedPermIDs = append(bannedPermIDs, strconv.Itoa(perm.ID))
			}
		}
	}
	return bannedPermIDs
}
func (s *TeamService) Delete(id int) error {
	return s.repo.Delete(id)
}