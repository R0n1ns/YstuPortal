package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/R0n1ns/YstuPortal/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserStorage struct {
	pgPool *pgxpool.Pool
}

func NewUserStorage(ctx context.Context, databaseURL string) (*UserStorage, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return &UserStorage{pgPool: pool}, nil
}

func (u *UserStorage) Close() {
	u.pgPool.Close()
}

func (u *UserStorage) GetUser(ctx context.Context, userName string) (*domain.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	newUser := domain.User{}
	var userID uuid.UUID
	query := `
       SELECT id, firstname, lastname, COALESCE(patronymic, ''), username,
              COALESCE(mail, ''), registered, COALESCE("group", ''), role
        FROM users
        WHERE username = $1;
`
	err := u.pgPool.QueryRow(ctx, query, userName).Scan(
		&userID,
		&newUser.FirstName,
		&newUser.LastName,
		&newUser.Patronymic,
		&newUser.UserName,
		&newUser.Mail,
		&newUser.Registered,
		&newUser.Group,
		&newUser.Role,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	query = `
       SELECT g.semester, g.course, s.title, g.type_of_control, sc.zed, g.evaluation, g.score, g.diploma
        FROM grades g
        JOIN subjects s ON s.id = g.subject_id
        LEFT JOIN subject_control sc
            ON sc.subject_id = g.subject_id AND sc.type_of_control = g.type_of_control
        WHERE g.user_id = $1;
`
	userRow, err := u.pgPool.Query(ctx, query, userID)
	if err != nil {
		return &newUser, err
	}
	defer userRow.Close()

	subjects := make([]domain.Subject, 0)
	for userRow.Next() {
		subj := domain.Subject{}
		if err := userRow.Scan(
			&subj.Semester,
			&subj.Course,
			&subj.Title,
			&subj.TypeOfControl,
			&subj.Zed,
			&subj.Evaluation,
			&subj.Mark,
			&subj.Diploma,
		); err != nil {
			return nil, fmt.Errorf("scan grade: %w", err)
		}
		subjects = append(subjects, subj)
	}
	if err := userRow.Err(); err != nil {
		return nil, fmt.Errorf("iterate grades: %w", err)
	}
	newUser.Grades = subjects
	return &newUser, nil
}

func (u *UserStorage) SaveUser(ctx context.Context, user *domain.User) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if user == nil {
		return errors.New("user is required")
	}
	tx, err := u.pgPool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	role := user.Role
	if role == "" {
		role = "student"
	}
	query := `
       INSERT INTO users (firstname, lastname, patronymic, username, mail, registered, "group", role)
	   VALUES ($1, $2, NULLIF($3, ''), $4, NULLIF($5, ''), $6, NULLIF($7, ''), $8)
	   ON CONFLICT (username) DO UPDATE SET
	       firstname = EXCLUDED.firstname,
	       lastname = EXCLUDED.lastname,
	       patronymic = EXCLUDED.patronymic,
	       mail = COALESCE(EXCLUDED.mail, users.mail),
	       registered = EXCLUDED.registered,
	       "group" = EXCLUDED."group"
	   RETURNING id;
`
	var userID uuid.UUID
	err = tx.QueryRow(ctx, query,
		user.FirstName,
		user.LastName,
		user.Patronymic,
		user.UserName,
		user.Mail,
		user.Registered,
		user.Group,
		role,
	).Scan(&userID)
	if err != nil {
		return fmt.Errorf("ошибка создания пользователя: %w", err)
	}

	if user.Grades != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM grades WHERE user_id = $1`, userID); err != nil {
			return fmt.Errorf("replace grades: %w", err)
		}
		querySubject := `
       INSERT INTO subjects (title, description)
       VALUES ($1, $2)
	   ON CONFLICT (title) DO UPDATE SET title = EXCLUDED.title
       RETURNING id;
`
		querySubjectControl := `
       INSERT INTO subject_control (subject_id, type_of_control, zed)
       VALUES ($1, $2, $3)
       ON CONFLICT (subject_id, type_of_control) DO UPDATE SET zed = EXCLUDED.zed;
`
		queryGrade := `
       INSERT INTO grades (user_id, subject_id, semester, course, type_of_control, score, evaluation, diploma)
       VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
       ON CONFLICT (user_id, subject_id, semester, course, type_of_control)
       DO UPDATE SET score = EXCLUDED.score, evaluation = EXCLUDED.evaluation, diploma = EXCLUDED.diploma;
`
		for _, subject := range user.Grades {
			var subjectID uuid.UUID
			err = tx.QueryRow(ctx, querySubject, subject.Title, "").Scan(&subjectID)
			if err != nil {
				return fmt.Errorf("ошибка создания предмета: %w", err)
			}
			_, err = tx.Exec(ctx, querySubjectControl, subjectID, subject.TypeOfControl, subject.Zed)
			if err != nil {
				return fmt.Errorf("ошибка создания контроля: %w", err)
			}
			_, err = tx.Exec(ctx, queryGrade,
				userID,
				subjectID,
				subject.Semester,
				subject.Course,
				subject.TypeOfControl,
				subject.Mark,
				subject.Evaluation,
				subject.Diploma,
			)
			if err != nil {
				return fmt.Errorf("ошибка создания оценок: %w", err)
			}
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("ошибка фиксации транзакции: %w", err)
	}
	return nil
}
