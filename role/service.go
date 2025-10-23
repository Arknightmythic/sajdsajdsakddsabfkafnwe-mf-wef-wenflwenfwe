package role

import (
	"dokuprime-be/permission"
	"dokuprime-be/team"
	"strconv"
)

type RoleService struct {
	repoRole       *RoleRepository
	repoTeam       *team.TeamRepository
	repoPermission *permission.PermissionRepository
}

func NewRoleService(repoRole *RoleRepository, repoTeam *team.TeamRepository, repoPermission *permission.PermissionRepository) *RoleService {
	return &RoleService{
		repoRole:       repoRole,
		repoTeam:       repoTeam,
		repoPermission: repoPermission,
	}
}

func (s *RoleService) Create(role Role) error {
	return s.repoRole.Create(role)
}

func (s *RoleService) GetAll() ([]GetRoleDTO, error) {
	roles, err := s.repoRole.GetAll()
	if err != nil {
		return nil, err
	}

	var getRolesDto []GetRoleDTO

	for _, role := range roles {
		team, err := s.repoTeam.GetByID(role.TeamID)
		if err != nil {
			return nil, err
		}

		var permissionsDto []permission.Permission

		for _, permission := range role.Permissions {
			permissionID, err := strconv.Atoi(permission)
			if err != nil {
				return nil, err
			}

			permissionDto, err := s.repoPermission.GetByID(permissionID)
			if err != nil {
				return nil, err
			}

			permissionsDto = append(permissionsDto, *permissionDto)
		}

		getRoleDto := GetRoleDTO{
			ID:          role.ID,
			Name:        role.Name,
			Permissions: permissionsDto,
			Team:        *team,
		}

		getRolesDto = append(getRolesDto, getRoleDto)
	}

	return getRolesDto, nil
}

func (s *RoleService) GetByID(id int) (*GetRoleDTO, error) {
	role, err := s.repoRole.GetByID(id)
	if err != nil {
		return nil, err
	}

	team, err := s.repoTeam.GetByID(role.TeamID)
	if err != nil {
		return nil, err
	}

	var permissionsDto []permission.Permission

	for _, permission := range role.Permissions {
		permissionID, err := strconv.Atoi(permission)
		if err != nil {
			return nil, err
		}

		permissionDto, err := s.repoPermission.GetByID(permissionID)
		if err != nil {
			return nil, err
		}

		permissionsDto = append(permissionsDto, *permissionDto)
	}

	getRoleDto := &GetRoleDTO{
		ID:          role.ID,
		Name:        role.Name,
		Permissions: permissionsDto,
		Team:        *team,
	}

	return getRoleDto, nil
}

func (s *RoleService) Update(id int, role Role) error {
	return s.repoRole.Update(id, role)
}

func (s *RoleService) Delete(id int) error {
	return s.repoRole.Delete(id)
}
