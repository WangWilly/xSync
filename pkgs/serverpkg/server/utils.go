package server

import (
	"strings"

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
)

// getAllUsers retrieves all users from the database
func (s *Server) getAllUsers() ([]*model.User, error) {
	var users []*model.User
	err := s.db.Select(&users, "SELECT * FROM users ORDER BY screen_name")
	return users, err
}

// getUserEntities retrieves all entities for a specific user
func (s *Server) getUserEntities(userID uint64) ([]*model.UserEntity, error) {
	var entities []*model.UserEntity
	err := s.db.Select(&entities, "SELECT * FROM user_entities WHERE user_id = ? ORDER BY name", userID)
	return entities, err
}

// convertToRelativePath converts absolute media paths to relative paths for serving
func (s *Server) convertToRelativePath(absolutePath string) string {
	// Find the "conf/users/" part in the absolute path
	usersIndex := strings.Index(absolutePath, "conf/users/")
	if usersIndex == -1 {
		// If "conf/users/" is not found, try to extract from the end
		// This handles cases where the path might be structured differently
		pathParts := strings.Split(absolutePath, "/")
		for i, part := range pathParts {
			if part == "users" && i > 0 && pathParts[i-1] == "conf" {
				// Join everything after "users/"
				if i+1 < len(pathParts) {
					return strings.Join(pathParts[i+1:], "/")
				}
			}
		}
		// If still not found, return the original path
		return absolutePath
	}

	// Extract the relative path after "conf/users/"
	relativePath := absolutePath[usersIndex+len("conf/users/"):]
	return relativePath
}
