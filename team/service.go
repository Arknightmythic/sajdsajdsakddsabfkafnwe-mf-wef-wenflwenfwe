package team


import (
	"dokuprime-be/permission" // Import permission repo
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
	// 1. Ambil data Team LAMA dari database sebelum di-update
	oldTeam, err := s.repo.GetByID(team.ID)
	if err != nil {
		return err
	}

	// 2. Deteksi Page apa saja yang DIHAPUS (Unselected)
	// Buat map dari page baru untuk pengecekan cepat
	newPagesMap := make(map[string]bool)
	for _, p := range team.Pages {
		newPagesMap[p] = true
	}

	var removedPages []string
	for _, p := range oldTeam.Pages {
		if !newPagesMap[p] {
			removedPages = append(removedPages, p)
		}
	}

	// 3. Jika ada page yang dihapus, kita harus bersihkan permission di Role
	if len(removedPages) > 0 {
		// Ambil semua master permission untuk mencocokkan nama dengan ID
		allPerms, err := s.repoPermission.GetAll()
		if err != nil {
			return err
		}

		var bannedPermIDs []string
		
		// Cari ID permission yang namanya diawali dengan page yang dihapus
		// Contoh: Page "document-management" dihapus -> cari permission "document-management:..."
		for _, perm := range allPerms {
			for _, page := range removedPages {
				// Gunakan ":" agar tidak salah match (misal "user" tidak menghapus "user-management")
				prefix := page + ":" 
				if strings.HasPrefix(perm.Name, prefix) {
					bannedPermIDs = append(bannedPermIDs, strconv.Itoa(perm.ID))
				}
			}
		}

		// Eksekusi pembersihan Role jika ada ID yang harus dihapus
		if len(bannedPermIDs) > 0 {
			if err := s.repo.RevokeRolePermissions(team.ID, bannedPermIDs); err != nil {
				return err
			}
		}
	}

	// 4. Update Team seperti biasa
	return s.repo.Update(team)
}

func (s *TeamService) Delete(id int) error {
	return s.repo.Delete(id)
}