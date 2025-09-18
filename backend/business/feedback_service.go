package business

import (
	"context"
	"fmt"
	"log"
	"strings"

	sqlc "github.com/greenwaltc/kellogg-music-match/backend/db/sqlc"
)

type FeedbackService struct {
	userRepo UserRepository
}

func NewFeedbackService(userRepo UserRepository) *FeedbackService {
	return &FeedbackService{
		userRepo: userRepo,
	}
}

func (fs *FeedbackService) SubmitFeedback(ctx context.Context, username string, feedbackText string) (*sqlc.Feedback, error) {
	// Validate feedback text
	feedbackText = strings.TrimSpace(feedbackText)
	if feedbackText == "" {
		return nil, fmt.Errorf("feedback text cannot be empty")
	}

	if len(feedbackText) > 1000 {
		return nil, fmt.Errorf("feedback text cannot exceed 1000 characters")
	}

	// Get user by username
	user, err := fs.userRepo.GetUserByUsername(ctx, username)
	if err != nil {
		log.Printf("Error retrieving user %s: %v", username, err)
		return nil, fmt.Errorf("user not found")
	}

	// Create feedback
	feedback, err := fs.userRepo.CreateFeedback(ctx, user.ID, feedbackText)
	if err != nil {
		log.Printf("Error creating feedback for user %s: %v", username, err)
		return nil, fmt.Errorf("failed to submit feedback")
	}

	log.Printf("Feedback submitted successfully by user %s", username)
	return feedback, nil
}

func (fs *FeedbackService) GetUserFeedback(ctx context.Context, username string) ([]sqlc.Feedback, error) {
	// Get user by username
	user, err := fs.userRepo.GetUserByUsername(ctx, username)
	if err != nil {
		log.Printf("Error retrieving user %s: %v", username, err)
		return nil, fmt.Errorf("user not found")
	}

	// Get feedback for user
	feedback, err := fs.userRepo.GetFeedbackByUser(ctx, user.ID)
	if err != nil {
		log.Printf("Error retrieving feedback for user %s: %v", username, err)
		return nil, fmt.Errorf("failed to retrieve feedback")
	}

	return feedback, nil
}
