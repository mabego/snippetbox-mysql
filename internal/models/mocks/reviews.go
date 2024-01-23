package mocks

import "github.com/mabego/snippetbox-mysql/internal/models"

type ReviewModel struct{}

// newMockReview creates an instance of the Review struct with mock data.
func newMockReview() *models.Review {
	return &models.Review{
		Reviews:   5,
		UserID:    1,
		SnippetID: 1,
	}
}

func (m *ReviewModel) Insert(userID, snippetID int) error { return nil }

func (m *ReviewModel) Exists(userID, snippetID int) (bool, error) {
	return false, nil
}

func (m *ReviewModel) Get(userID, snippetID int) (*models.Review, error) {
	switch snippetID {
	case 1:
		return newMockReview(), nil
	default:
		return &models.Review{}, nil
	}
}

func (m *ReviewModel) Update(userID, snippetID int) error { return nil }
