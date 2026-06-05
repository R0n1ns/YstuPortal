CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

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

