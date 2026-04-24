package auth

import "github.com/labstack/echo/v5"

const CurrentUserContextKey = "auth.current_user"

type CurrentUser struct {
	UserID int64
	Email  string
	Roles  []string
}

func SetCurrentUser(c *echo.Context, user CurrentUser) {
	c.Set(CurrentUserContextKey, CurrentUser{
		UserID: user.UserID,
		Email:  user.Email,
		Roles:  append([]string(nil), user.Roles...),
	})
}

func GetCurrentUser(c *echo.Context) (CurrentUser, bool) {
	value := c.Get(CurrentUserContextKey)
	if value == nil {
		return CurrentUser{}, false
	}

	user, ok := value.(CurrentUser)
	if !ok {
		return CurrentUser{}, false
	}

	return user, true
}

func HasRole(user CurrentUser, role string) bool {
	for _, currentRole := range user.Roles {
		if currentRole == role {
			return true
		}
	}

	return false
}

func HasAnyRole(user CurrentUser, roles ...string) bool {
	for _, role := range roles {
		if HasRole(user, role) {
			return true
		}
	}

	return false
}
