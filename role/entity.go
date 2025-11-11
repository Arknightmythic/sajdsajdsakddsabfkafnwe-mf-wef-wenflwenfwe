package role

import (
	"dokuprime-be/permission"
	"dokuprime-be/team"

	"github.com/lib/pq"
)

type Role struct {
	ID          int            `db:"id" json:"id"`
	Name        string         `db:"name" json:"name"`
	Permissions pq.StringArray `db:"permissions" json:"permissions"`
	TeamID      int            `db:"team_id" json:"team_id"`
}

type GetRoleDTO struct {
	ID          int                     `db:"id" json:"id"`
	Name        string                  `db:"name" json:"name"`
	Permissions []permission.Permission `db:"permissions" json:"permissions"`
	Team        team.Team               `db:"team" json:"team"`
}
