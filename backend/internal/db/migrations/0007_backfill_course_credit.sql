-- Give courses created before the curriculum existed their credit, and move the
-- seeded demo course onto its real syllabus code.
UPDATE courses c
JOIN course_catalog cc ON cc.code = c.code
SET c.credit = cc.credit, c.catalog_id = cc.id;

UPDATE courses
SET code = 'CSE 3103',
    title = 'Compiler Design',
    credit = 3,
    catalog_id = (SELECT id FROM course_catalog WHERE code = 'CSE 3103')
WHERE code = 'CSC301'
  AND NOT EXISTS (
    SELECT 1 FROM (SELECT session_id FROM courses WHERE code = 'CSE 3103') other
    WHERE other.session_id = courses.session_id
  );
