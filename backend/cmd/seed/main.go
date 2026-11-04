// Command seed fills a fresh database with a realistic demonstration cohort.
// Safe to re-run: it keyed everything off fixed identifiers.
package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/istiyak/secure-exam-eval/internal/api"
	"github.com/istiyak/secure-exam-eval/internal/auth"
	"github.com/istiyak/secure-exam-eval/internal/config"
	"github.com/istiyak/secure-exam-eval/internal/db"
	"github.com/istiyak/secure-exam-eval/internal/storage"
	"github.com/istiyak/secure-exam-eval/internal/store"
)

type person struct {
	email, name, role string
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	conn, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	if err := db.Migrate(ctx, conn); err != nil {
		log.Fatal(err)
	}
	st := store.New(conn)
	files, err := storage.New(cfg.StorageDriver, cfg.StorageDir, cfg.S3Bucket, cfg.S3Region, cfg.S3Endpoint)
	if err != nil {
		log.Fatal(err)
	}

	pw := os.Getenv("SEED_PASSWORD")
	if pw == "" {
		pw = "ExamOffice#2026"
	}

	people := []person{
		{"registrar@uni.test", "Bilkis Rahman", "HALL_MANAGER"},
		{"examiner.one@uni.test", "Dr. Tariq Hossain", "EVALUATOR"},
		{"examiner.two@uni.test", "Dr. Priya Nair", "EVALUATOR"},
		{"examiner.three@uni.test", "Prof. Samuel Okoro", "EVALUATOR"},
	}
	ids := map[string]string{}
	for _, p := range people {
		u, err := ensureUser(ctx, st, p, pw)
		if err != nil {
			log.Fatalf("%s: %v", p.email, err)
		}
		ids[p.email] = u
	}

	ses, err := ensureSession(ctx, st)
	if err != nil {
		log.Fatal(err)
	}
	crs, err := ensureCourse(ctx, st, ses, ids["examiner.one@uni.test"])
	if err != nil {
		log.Fatal(err)
	}

	roster := []struct {
		no, name, email string
		m1, m2          float64
		third           float64
	}{
		{"MAT/2026/0101", "Adaeze Nwosu", "adaeze@student.test", 62, 68, 0},
		{"MAT/2026/0102", "Ibrahim Sule", "ibrahim@student.test", 41, 58, 61},
		{"MAT/2026/0103", "Chen Wei", "chen@student.test", 55, 72, 70},
		{"MAT/2026/0104", "Fatima Zahra", "fatima@student.test", 38, 79, 64},
		{"MAT/2026/0105", "Kwame Mensah", "kwame@student.test", 71, 52, 58},
	}

	for i, r := range roster {
		stu, su, err := ensureStudent(ctx, st, r.no, r.name, r.email, ses, pw)
		if err != nil {
			log.Fatalf("%s: %v", r.name, err)
		}
		code := fmt.Sprintf("CSC301-26-%04d", i+1)
		var exists int
		_ = conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM scripts WHERE script_code = ?`, code).Scan(&exists)
		if exists > 0 {
			continue
		}

		pdf := buildPDF(r.name, code, "CSC301 — Compilers")
		key := "scripts/csc301/" + code + ".pdf"
		if err := files.Put(ctx, key, bytes.NewReader(pdf), int64(len(pdf)), "application/pdf"); err != nil {
			log.Fatal(err)
		}
		pages := 1
		sc, err := st.CreateScript(ctx, store.NewScript{
			Code: code, CourseID: crs, StudentID: stu, PDFKey: key,
			SHA256: "seed", Bytes: int64(len(pdf)), Pages: &pages,
		})
		if err != nil {
			log.Fatalf("%s: %v", code, err)
		}
		if err := st.AssignPair(ctx, sc.ID, ids["examiner.one@uni.test"], ids["examiner.two@uni.test"], ""); err != nil {
			log.Fatal(err)
		}
		if _, err := st.SubmitMark(ctx, sc.ID, ids["examiner.one@uni.test"], r.m1, nil); err != nil {
			log.Fatal(err)
		}
		dec, err := st.SubmitMark(ctx, sc.ID, ids["examiner.two@uni.test"], r.m2, nil)
		if err != nil {
			log.Fatal(err)
		}
		if dec.NeedThird {
			if err := st.AssignThird(ctx, sc.ID, ids["examiner.three@uni.test"], ""); err != nil {
				log.Fatal(err)
			}
			if _, err := st.SubmitMark(ctx, sc.ID, ids["examiner.three@uni.test"], r.third, nil); err != nil {
				log.Fatal(err)
			}
		}
		_ = su
	}

	_ = api.NewLimiter
	fmt.Println("seed complete")
	fmt.Printf("admin    : %s / %s\n", cfg.BootstrapEmail, cfg.BootstrapPass)
	fmt.Printf("registrar: registrar@uni.test / %s\n", pw)
	fmt.Printf("examiner : examiner.one@uni.test / %s\n", pw)
	fmt.Printf("student  : adaeze@student.test / %s\n", pw)
}

func ensureUser(ctx context.Context, st *store.Store, p person, pw string) (string, error) {
	u, err := st.UserByEmail(ctx, p.email)
	if err == nil {
		return u.ID, nil
	}
	hash, err := auth.HashPassword(pw)
	if err != nil {
		return "", err
	}
	created, err := st.CreateUser(ctx, p.email, hash, p.name, p.role)
	if err != nil {
		return "", err
	}
	return created.ID, nil
}

func ensureSession(ctx context.Context, st *store.Store) (string, error) {
	var id string
	err := st.DB.QueryRowContext(ctx, `SELECT id FROM academic_sessions WHERE name = '2026/2027'`).Scan(&id)
	if err == nil {
		return id, nil
	}
	s, err := st.CreateSession(ctx, "2026/2027", "A26")
	if err != nil {
		return "", err
	}
	return s.ID, nil
}

func ensureCourse(ctx context.Context, st *store.Store, ses, instructor string) (string, error) {
	var id string
	err := st.DB.QueryRowContext(ctx, `SELECT id FROM courses WHERE code = 'CSC301' AND session_id = ?`, ses).Scan(&id)
	if err == nil {
		return id, nil
	}
	c, err := st.CreateCourse(ctx, "CSC301", "Compilers", ses, 100, 40, &instructor)
	if err != nil {
		return "", err
	}
	return c.ID, nil
}

func ensureStudent(ctx context.Context, st *store.Store, no, name, email, ses, pw string) (string, string, error) {
	rows, _ := st.ListStudents(ctx, no, ses)
	if len(rows) > 0 {
		return rows[0].ID, "", nil
	}
	hash, err := auth.HashPassword(pw)
	if err != nil {
		return "", "", err
	}
	u, err := st.CreateUser(ctx, email, hash, name, "STUDENT")
	if err != nil {
		return "", "", err
	}
	s, err := st.CreateStudent(ctx, no, name, &email, ses)
	if err != nil {
		return "", "", err
	}
	if err := st.LinkStudentUser(ctx, s.ID, u.ID); err != nil {
		return "", "", err
	}
	return s.ID, u.ID, nil
}

// buildPDF writes a small but structurally valid single-page PDF.
func buildPDF(name, code, course string) []byte {
	var b bytes.Buffer
	var offs []int
	b.WriteString("%PDF-1.4\n")
	add := func(s string) {
		offs = append(offs, b.Len())
		b.WriteString(s)
	}
	add("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")
	add("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	add("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 5 0 R >> >> /Contents 4 0 R >>\nendobj\n")
	stream := fmt.Sprintf("BT /F1 18 Tf 70 760 Td (%s) Tj ET BT /F1 12 Tf 70 730 Td (Cover code: %s) Tj ET BT /F1 11 Tf 70 700 Td (Candidate name removed before marking.) Tj ET", course, code)
	add(fmt.Sprintf("4 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n", len(stream), stream))
	add("5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n")

	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(offs)+1)
	for _, o := range offs {
		fmt.Fprintf(&b, "%010d 00000 n \n", o)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offs)+1, xref)
	_ = name
	_ = time.Now()
	return b.Bytes()
}
