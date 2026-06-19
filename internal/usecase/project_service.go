package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/achmichael/pribadi-go/internal/repository/sqlc"
)

type ProjectService interface {
	HandleCommand(ctx context.Context, sender, command string) (string, error)
}

type projectService struct {
	repo repository.Repository
}

func NewProjectService(repo repository.Repository) ProjectService {
	return &projectService{repo}
}

func (s *projectService) HandleCommand(ctx context.Context, sender, command string) (string, error) {
	parts := strings.Fields(command)
	if len(parts) < 2 {
		return "Invalid command", nil
	}

	switch parts[0] {
	case "/project":
		if parts[1] == "create" && len(parts) > 2 {
			_, err := s.repo.InsertProject(ctx, sqlc.InsertProjectParams{Name: parts[2], OwnerJid: sender})
			if err != nil {
				return "Error creating project", err
			}
			return "Project created", nil
		}
		if parts[1] == "list" {
			projects, _ := s.repo.ListProjects(ctx, 10, 0)
			var sb strings.Builder
			sb.WriteString("```\nID | Name\n---\n")
			for _, p := range projects {
				sb.WriteString(fmt.Sprintf("%d | %s\n", p.ID, p.Name))
			}
			sb.WriteString("```")
			return sb.String(), nil
		}
	}
	return "Command not found", nil
}
