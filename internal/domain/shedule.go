package domain

import (
	"context"

	"github.com/google/uuid"
)
import "time"

type Subject struct {
	Id            uuid.UUID `json:"id"`
	Course        int       `json:"course"`
	Semester      int       `json:"semester"`
	Title         string    `json:"title"`
	TypeOfControl string    `json:"typeofcontrol"`
	Zed           int       `json:"zed"`
	Mark          string    `json:"mark"`
	Evaluation    string    `json:"evaluation"`
	Diploma       bool      `json:"diploma"`
	Teacher       string    `json:"teacher"`
}

type Lesson struct {
	Id              uuid.UUID `json:"id"`
	Start           time.Time `json:"start"`
	End             time.Time `json:"end"`
	DurationLessons int       `json:"duration"`
	Title           string    `json:"title"`
	Subject         Subject   `json:"subject"`
	Description     string    `json:"description"`
	Room            string    `json:"room"`
	Teacher         string    `json:"teacher"`
}

type LessonPort interface {
	GetLessonsByDate(ctx context.Context, start, end time.Time) []Lesson
}
