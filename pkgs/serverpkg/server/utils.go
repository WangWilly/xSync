package server

import (
	"strings"
)

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
