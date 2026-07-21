package domain

type Subject struct {
	Course        int    `json:"course"`
	Semester      int    `json:"semester"`
	Title         string `json:"title"`
	TypeOfControl string `json:"type_of_control"`
	Zed           int    `json:"zed"`
	Mark          string `json:"mark"`
	Evaluation    string `json:"evaluation"`
	Diploma       bool   `json:"diploma"`
	Teacher       string `json:"teacher,omitempty"`
}
