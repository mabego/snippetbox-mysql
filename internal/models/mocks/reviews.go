package mocks

import "github.com/mabego/snippetbox-mysql/internal/models"

type ReviewModel struct{}

const reviews = 5

// newMockReview creates an instance of the Review struct with mock data.
func newMockReview() *models.Review {
	return &models.Review{
		UserID:    1,
		SnippetID: 1,
		Reviews:   reviews,
	}
}

func (m *ReviewModel) Insert(_, _ int) error { return nil }

func (m *ReviewModel) Exists(_, _ int) (bool, error) { return false, nil }

func (m *ReviewModel) Get(_, snippetID int) (*models.Review, error) {
	switch snippetID {
	case 1:
		return newMockReview(), nil
	default:
		return &models.Review{}, nil
	}
}

func (m *ReviewModel) Update(_, _ int) error { return nil }
