package role

// Role — примитивная ролевая модель сервиса (ЛР4).
// В БД роль хранится флагом battery_app_users.is_moderator,
// а в JWT попадает числовым значением.
type Role int

const (
	// User — обычный пользователь: может создавать, публиковать,
	// удалять свои услуги и ставить лайки.
	User Role = iota
	// Moderator — пользователь с расширенными правами:
	// дополнительно может удалять чужие услуги.
	Moderator
)

func (r Role) String() string {
	switch r {
	case Moderator:
		return "moderator"
	default:
		return "user"
	}
}

// FromModeratorFlag переводит системный флаг БД в роль.
func FromModeratorFlag(isModerator bool) Role {
	if isModerator {
		return Moderator
	}
	return User
}

// IsValid проверяет, что роль известна приложению.
func (r Role) IsValid() bool {
	return r == User || r == Moderator
}
