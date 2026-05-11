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
	_, err = pool.Exec(context.Background(), "-- PostgreSQL DDL for User / Subject / Lesson\n\nCREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";\n\nCREATE TABLE IF NOT EXISTS users (\nid uuid PRIMARY KEY DEFAULT uuid_generate_v4(),\nfirstname text NOT NULL,\nlastname text NOT NULL,\npatronymic text,\nusername text NOT NULL UNIQUE,\nmail text NOT NULL UNIQUE,\npassword_hash text NOT NULL,\nregistered boolean NOT NULL DEFAULT false,\n\"group\" text\n);\n\nCREATE TABLE IF NOT EXISTS subjects (\nid uuid PRIMARY KEY DEFAULT uuid_generate_v4(),\ntitle text NOT NULL,\nteacher text\n);\n\nCREATE TABLE IF NOT EXISTS lessons (\nid uuid PRIMARY KEY DEFAULT uuid_generate_v4(),\nstart_time timestamptz NOT NULL,\nend_time timestamptz NOT NULL,\nduration int NOT NULL,\ntitle text NOT NULL,\nsubject_id uuid NOT NULL REFERENCES subjects(id) ON DELETE RESTRICT,\ndescription text,\nroom text,\nteacher text\n);\n\n-- расширенные оценки пользователя\nCREATE TABLE IF NOT EXISTS user_estimations (\nid uuid PRIMARY KEY DEFAULT uuid_generate_v4(),\nuser_name text NOT NULL REFERENCES users(username) ON DELETE CASCADE,\nnumber int NOT NULL DEFAULT 0,\ncourse int NOT NULL DEFAULT 0,\nsemester int NOT NULL DEFAULT 0,\ntitle text NOT NULL DEFAULT '',\ntype_of_control text NOT NULL DEFAULT '',\nzed int NOT NULL DEFAULT 0,\nscore text NOT NULL DEFAULT '',\nevaluation text NOT NULL DEFAULT '',\ndiploma boolean NOT NULL DEFAULT false\n);\n\n-- полезные индексы\nCREATE INDEX IF NOT EXISTS idx_lessons_start_time ON lessons(start_time);\nCREATE INDEX IF NOT EXISTS idx_user_estimations_user ON user_estimations(user_name);")
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
