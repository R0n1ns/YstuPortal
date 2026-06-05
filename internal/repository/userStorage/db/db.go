package db

import (
	"YstuPortal/internal/domain"
	"context"
	"fmt"

	"github.com/google/uuid"
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
	var userID uuid.UUID
	query := `
       SELECT id, firstname, lastname, patronymic, username, mail, password_hash, registered, "group", role
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
		&newUser.Password,
		&newUser.Registered,
		&newUser.Group,
		&newUser.Role,
	)
	if err != nil {
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
		err = userRow.Scan(
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
		subjects = append(subjects, subj)
	}
	newUser.Estimations = subjects
	return &newUser, nil
}
func (u *UserStorage) SaveUser(ctx context.Context, user *domain.User) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	//u.mu.Lock()
	//u.users[user.UserName] = user
	//u.mu.Unlock()
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
       INSERT INTO users (firstname, lastname, patronymic, username, mail, password_hash, registered, "group", role)
       VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
       RETURNING id;
`
	var userID uuid.UUID
	err = tx.QueryRow(ctx, query,
		user.FirstName,
		user.LastName,
		user.Patronymic,
		user.UserName,
		user.Mail,
		user.Password,
		user.Registered,
		user.Group,
		role,
	).Scan(&userID)
	if err != nil {
		return fmt.Errorf("ошибка создания пользователя: %w", err)
	}

	if user.Estimations != nil {
		querySubject := `
       INSERT INTO subjects (title, description)
       VALUES ($1, $2)
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
		for _, subject := range user.Estimations {
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

//TODO: сделать сохранение пользователя в бд и для быстрокого получения хеширования в redis
