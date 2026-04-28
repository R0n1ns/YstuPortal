package domain

import (
	"context"

	"github.com/google/uuid"
)
import "time"

type Lesson struct {
	Id              uuid.UUID `json:"id"`
	Start           time.Time `json:"start"`
	End             time.Time `json:"end"`
	DurationLessons int       `json:"duration"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	Room            string    `json:"room"`
	Teacher         string    `json:"teacher"`
}

type LessonPort interface {
	GetLessonsByDate(ctx context.Context, start, end time.Time) []Lesson
}
