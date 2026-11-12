// Package models contains API-facing row types.
package models

import "time"

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type Session struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// CatalogCourse is one entry of the fixed university curriculum.
type CatalogCourse struct {
	ID       string  `json:"id"`
	Code     string  `json:"code"`
	Title    string  `json:"title"`
	Credit   float64 `json:"credit"`
	Term     string  `json:"term"`
	Category string  `json:"category"`
}

type Student struct {
	ID             string    `json:"id"`
	StudentNo      string    `json:"student_no"`
	RegistrationNo *string   `json:"registration_no,omitempty"`
	FullName       string    `json:"full_name"`
	Email          *string   `json:"email,omitempty"`
	SessionID      string    `json:"session_id"`
	Session        string    `json:"session,omitempty"`
	UserID         *string   `json:"user_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type Course struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Title       string    `json:"title"`
	Credit      float64   `json:"credit"`
	SessionID   string    `json:"session_id"`
	Session     string    `json:"session,omitempty"`
	MaxMarks    float64   `json:"max_marks"`
	PassMarks   float64   `json:"pass_marks"`
	Instructor  *string   `json:"instructor_name,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	ScriptCount int       `json:"script_count"`
}

// Mark is one evaluator's submission. Slot hides which examiner is which.
type Mark struct {
	Slot          string    `json:"slot"`
	Score         float64   `json:"score"`
	Comment       *string   `json:"comment,omitempty"`
	EvaluatorName string    `json:"evaluator_name,omitempty"`
	SubmittedAt   time.Time `json:"submitted_at"`
}

// ScriptBlind is what an evaluator sees: no student identity, only the code.
type ScriptBlind struct {
	ID          string    `json:"id"`
	Code        string    `json:"script_code"`
	Status      string    `json:"status"`
	CourseCode  string    `json:"course_code"`
	CourseTitle string    `json:"course_title"`
	MaxMarks    float64   `json:"max_marks"`
	PageCount   *int      `json:"page_count,omitempty"`
	MySlot      string    `json:"my_slot"`
	Marks       []Mark    `json:"marks"`
	FinalMarks  *float64  `json:"final_marks,omitempty"`
	AssignedAt  time.Time `json:"assigned_at"`
}

// Script is the full staff/admin view including the identity unmasking.
type Script struct {
	ID           string     `json:"id"`
	Code         string     `json:"script_code"`
	Status       string     `json:"status"`
	CourseID     string     `json:"course_id"`
	CourseCode   string     `json:"course_code"`
	CourseTitle  string     `json:"course_title"`
	MaxMarks     float64    `json:"max_marks"`
	StudentID    string     `json:"student_id"`
	StudentNo    string     `json:"student_no"`
	StudentName  string     `json:"student_name"`
	SessionName  string     `json:"session"`
	PageCount    *int       `json:"page_count,omitempty"`
	PDFBytes     int64      `json:"pdf_bytes"`
	Marks        []Mark     `json:"marks"`
	FinalMarks   *float64   `json:"final_marks,omitempty"`
	Gap          *float64   `json:"gap,omitempty"`
	Method       *string    `json:"method,omitempty"`
	RegisteredAt time.Time  `json:"registered_at"`
	ResolvedAt   *time.Time `json:"resolved_at,omitempty"`
}

// ResultRow powers the search screen and the student portal.
type ResultRow struct {
	ScriptID     string     `json:"script_id"`
	Code         string     `json:"script_code"`
	StudentID    string     `json:"student_id"`
	StudentNo    string     `json:"student_no"`
	StudentName  string     `json:"student_name"`
	Session      string     `json:"session"`
	CourseCode   string     `json:"course_code"`
	CourseTitle  string     `json:"course_title"`
	Status       string     `json:"status"`
	FinalMarks   *float64   `json:"final_marks,omitempty"`
	Grade        *string    `json:"grade,omitempty"`
	Passed       *bool      `json:"passed,omitempty"`
	Marks        []Mark     `json:"marks,omitempty"`
	Challenge    *string    `json:"challenge_status,omitempty"`
	Revealed     bool       `json:"revealed"`
	ResolvedAt   *time.Time `json:"resolved_at,omitempty"`
	RegisteredAt time.Time  `json:"registered_at"`
}

type Challenge struct {
	ID           string     `json:"id"`
	ScriptID     string     `json:"script_id"`
	Code         string     `json:"script_code"`
	StudentNo    string     `json:"student_no,omitempty"`
	StudentName  string     `json:"student_name,omitempty"`
	CourseCode   string     `json:"course_code,omitempty"`
	Reason       string     `json:"reason"`
	Status       string     `json:"status"`
	DecisionNote *string    `json:"decision_note,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	DecidedAt    *time.Time `json:"decided_at"`
}

type AuditEntry struct {
	ID        int64     `json:"id"`
	ActorName *string   `json:"actor_name"`
	ActorRole *string   `json:"actor_role"`
	Action    string    `json:"action"`
	Entity    string    `json:"entity"`
	EntityID  *string   `json:"entity_id"`
	Meta      any       `json:"meta"`
	CreatedAt time.Time `json:"created_at"`
}

type Stats struct {
	Scripts    map[string]int `json:"scripts_by_status"`
	Users      map[string]int `json:"users_by_role"`
	Challenges int            `json:"open_challenges"`
	Resolved   int            `json:"results_resolved"`
	ThirdRate  float64        `json:"third_review_rate"`
	Averages   float64        `json:"average_final_marks"`
}
