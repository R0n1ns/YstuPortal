package db

import (
	"YstuPortal/internal/domain"
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserStorage struct {
	//mu     sync.RWMutex
	pgPool *pgxpool.Pool
	//users  map[string]*domain.User
}

func NewUserStorage(pgUrl string) *UserStorage {

	pool, err := pgxpool.New(context.Background(), pgUrl)
	if err != nil {
		panic(fmt.Errorf("неправильные настройки подключения к бд %s", err))
	}
	err = pool.Ping(context.Background())
	if err != nil {
		panic(fmt.Errorf("не удалось подлкючиться к бд %s", err))
	}
	_, err = pool.Exec(context.Background(), `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users with role + role-specific nullable fields
CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    firstname text NOT NULL,
    lastname text NOT NULL,
    patronymic text,
    username text NOT NULL UNIQUE,
    mail text NOT NULL UNIQUE,
    password_hash text NOT NULL,
    registered boolean NOT NULL DEFAULT false,
    role text NOT NULL CHECK (role IN ('student', 'teacher', 'admin')),
    "group" text,
    course int,
    academic_title text,
    department text
);

CREATE TABLE subjects (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    title text NOT NULL,
    description text
);

CREATE TABLE subject_control (
    subject_id uuid NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    type_of_control text NOT NULL,
    zed int NOT NULL DEFAULT 0,
    PRIMARY KEY (subject_id, type_of_control)
);

CREATE TABLE lessons (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    subject_id uuid NOT NULL REFERENCES subjects(id) ON DELETE RESTRICT,
    teacher_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    start_time timestamptz NOT NULL,
    end_time timestamptz NOT NULL,
    room text
);

CREATE TABLE enrollments (
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subject_id uuid NOT NULL REFERENCES subjects(id) ON DELETE RESTRICT,
    PRIMARY KEY (user_id, subject_id)
);

CREATE TABLE grades (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subject_id uuid NOT NULL REFERENCES subjects(id) ON DELETE RESTRICT,
    semester int NOT NULL DEFAULT 0,
    course int NOT NULL DEFAULT 0,
    type_of_control text NOT NULL DEFAULT '',
    score text NOT NULL DEFAULT '',
    evaluation text NOT NULL DEFAULT '',
    diploma boolean NOT NULL DEFAULT false,
    FOREIGN KEY (subject_id, type_of_control)
        REFERENCES subject_control(subject_id, type_of_control),
    UNIQUE (user_id, subject_id, semester, course, type_of_control)
);

CREATE INDEX IF NOT EXISTS idx_lessons_start_time ON lessons(start_time);
CREATE INDEX IF NOT EXISTS idx_lessons_teacher ON lessons(teacher_id);
CREATE INDEX IF NOT EXISTS idx_grades_user ON grades(user_id);
CREATE INDEX IF NOT EXISTS idx_grades_subject ON grades(subject_id);
CREATE INDEX IF NOT EXISTS idx_enrollments_user ON enrollments(user_id);
CREATE INDEX IF NOT EXISTS idx_enrollments_subject ON enrollments(subject_id);
`)
	if err != nil {
		panic(fmt.Errorf("ошибка во время создания таблиц бд %s", err))
	}

	return &UserStorage{
		//users:  make(map[string]*domain.User),
		pgPool: pool,
	}
}
func (u *UserStorage) Close() {
	u.pgPool.Close()
}

func (u *UserStorage) GetUser(ctx context.Context, userName string) (*domain.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	//u.mu.Lock()
	//user, ok := u.users[userName]
	//if !ok {
	//	return nil, fmt.Errorf("нет такого пользователя")
	//}
	//u.mu.Unlock()
	newUser := domain.User{}
	query := `
       SELECT firstname, lastname, patronymic, username, mail,password_hash,registered,"group"
        FROM users
        WHERE username = $1;
`
	userRow, err := u.pgPool.Query(ctx, query, userName)
	if err != nil {
		return nil, err
	}

	defer userRow.Close()
	for userRow.Next() {
		err = userRow.Scan(
			&newUser.FirstName,
			&newUser.LastName,
			&newUser.Patronymic,
			&newUser.UserName,
			&newUser.Mail,
			&newUser.Password,
			&newUser.Registered,
			&newUser.Group,
		)
		if err != nil {
			return nil, err
		}
	}
	if newUser.UserName == "" {
		return nil, fmt.Errorf("пользователь не неайден")
	}

	query = `
       SELECT number, semester, course, title, type_of_control, zed, evaluation, score, diploma
        FROM user_estimations
        WHERE user_name = $1;
`
	userRow, err = u.pgPool.Query(ctx, query, userName)
	if err != nil {
		return &newUser, err
	}

	defer userRow.Close()
	subject := []domain.Subject{}
	for userRow.Next() {
		subj := domain.Subject{}
		err = userRow.Scan(
			&subj.Mark,
			&subj.Semester,
			&subj.Course,
			&subj.Title,
			&subj.TypeOfControl,
			&subj.Zed,
			&subj.Evaluation,
			&subj.Mark,
			&subj.Diploma,
		)
		if err != nil {
			continue
		}
		subject = append(subject, subj)
	}
	newUser.Estimations = subject
	return &newUser, nil
}
func (u *UserStorage) SaveUser(ctx context.Context, user *domain.User) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	//u.mu.Lock()
	//u.users[user.UserName] = user
	//u.mu.Unlock()
	query := `
       INSERT INTO users (firstname, lastname, patronymic, username, mail,password_hash,registered,"group") 
       VALUES ($1, $2, $3, $4, $5,$6,$7,$8);
`
	_, err := u.pgPool.Query(ctx, query,
		user.FirstName,
		user.LastName,
		user.Patronymic,
		user.UserName,
		user.Mail,
		user.Password,
		user.Registered,
		user.Group)
	if err != nil {
		return fmt.Errorf("ошибка создания пользователя: %w", err)
	}
	if user.Estimations != nil {
		query := `
       INSERT INTO user_estimations (user_name, number, semester, course, title, type_of_control, zed, evaluation, score, diploma)
       VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);
`
		for number, subject := range user.Estimations {
			_, err = u.pgPool.Exec(ctx, query,
				user.UserName,
				number,
				subject.Semester,
				subject.Course,
				subject.Title,
				subject.TypeOfControl,
				subject.Zed,
				subject.Evaluation,
				subject.Mark,
				subject.Diploma,
			)
			if err != nil {
				return fmt.Errorf("ошибка создания оценок: %w", err)
			}
		}
	}
	return nil
}

//TODO: сделать сохранение пользователя в бд и для быстрокого получения хеширования в redis
