package mocks

import (
	"time"

	"github.com/mabego/snippetbox-mysql/internal/models"
)

type SnippetModel struct{}

// newMockSnippet creates an instance of the Snippet struct with mock data.
func newMockSnippet() *models.Snippet {
	return &models.Snippet{
		ID:      1,
		Title:   "An old silent pond",
		Content: "An old silent pond...",
		Created: time.Now(),
		Expires: time.Now(),
	}
}

func (m *SnippetModel) Insert(string, string, int) (int, error) {
	mockID := 2
	return mockID, nil
}

func (m *SnippetModel) Get(id int) (*models.Snippet, error) {
	switch id {
	case 1:
		return newMockSnippet(), nil
	default:
		return nil, models.ErrNoRecord
	}
}

func (m *SnippetModel) Latest() ([]*models.Snippet, error) {
	return []*models.Snippet{newMockSnippet()}, nil
}
